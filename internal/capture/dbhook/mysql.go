package dbhook

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
)

// MySQLHook is a MySQL protocol proxy that parses COM_QUERY packets to capture SQL statements
type MySQLHook struct {
	listenAddr  string
	targetAddr  string
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
	connsMu     sync.RWMutex
	activeConns map[*activeConn]struct{}
}

// MySQL protocol constants
const (
	mysqlComQuery       = 0x03
	mysqlComStmtPrepare = 0x16
	mysqlComStmtExecute = 0x17
)

func NewMySQLHook(listenAddr, targetAddr string) *MySQLHook {
	return &MySQLHook{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
		activeConns: make(map[*activeConn]struct{}),
	}
}

func (h *MySQLHook) Type() string { return "mysql" }

func (h *MySQLHook) SideEffects() <-chan model.SideEffect {
	return h.sideEffects
}

func (h *MySQLHook) Start(ctx context.Context) error {
	var err error
	h.listener, err = net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", "mysql", "listen", h.listenAddr, "target", h.targetAddr)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.done:
				return
			default:
			}

			conn, err := h.listener.Accept()
			if err != nil {
				select {
				case <-h.done:
					return
				default:
					logger.Error("mysql accept error", err)
					continue
				}
			}

			h.wg.Add(1)
			go func() {
				defer h.wg.Done()
				h.handleConn(conn)
			}()
		}
	}()

	return nil
}

func (h *MySQLHook) Flush(ctx context.Context) error {
	for _, conn := range h.snapshotActiveConns() {
		ack := make(chan struct{})
		select {
		case conn.flushCh <- ack:
		case <-conn.done:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}

		select {
		case <-ack:
		case <-conn.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *MySQLHook) Stop() error {
	close(h.done)
	if h.listener != nil {
		h.listener.Close()
	}
	h.wg.Wait()
	close(h.sideEffects)
	return nil
}

func (h *MySQLHook) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	// Connect to the real MySQL server
	serverConn, err := net.DialTimeout("tcp", h.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("mysql connect target failed", err)
		return
	}
	defer serverConn.Close()

	conn := &activeConn{
		client:  clientConn,
		server:  serverConn,
		flushCh: make(chan chan struct{}),
		done:    make(chan struct{}),
	}
	h.registerConn(conn)
	defer h.unregisterConn(conn)

	// Bidirectional forwarding while sniffing client-to-server data
	var wg sync.WaitGroup

	// Server -> Client (passthrough)
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()

	// Client -> Server (sniff MySQL packets)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(conn.done)
		h.sniffClientToServer(conn)
	}()

	wg.Wait()
}

// sniffClientToServer sniffs data sent by the client and parses MySQL protocol packets
func (h *MySQLHook) sniffClientToServer(conn *activeConn) {
	buf := make([]byte, 64*1024)
	for {
		select {
		case ack := <-conn.flushCh:
			h.flushConn(conn, buf)
			close(ack)
			continue
		default:
		}

		_ = conn.client.SetReadDeadline(time.Now().Add(hookReadPollInterval))
		n, err := conn.client.Read(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}

		// Forward to server
		if _, err := conn.server.Write(buf[:n]); err != nil {
			return
		}

		// Try to parse MySQL packet
		h.parseMySQLPacket(buf[:n])
	}
}

func (h *MySQLHook) flushConn(conn *activeConn, buf []byte) {
	idleDeadline := time.Now().Add(hookFlushIdleWindow)
	for {
		remaining := time.Until(idleDeadline)
		if remaining <= 0 {
			return
		}

		_ = conn.client.SetReadDeadline(time.Now().Add(remaining))
		n, err := conn.client.Read(buf)
		if n > 0 {
			if _, writeErr := conn.server.Write(buf[:n]); writeErr != nil {
				return
			}
			h.parseMySQLPacket(buf[:n])
			idleDeadline = time.Now().Add(hookFlushIdleWindow)
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return
			}
			return
		}
	}
}

// parseMySQLPacket parses a MySQL protocol packet and extracts SQL statements
func (h *MySQLHook) parseMySQLPacket(data []byte) {
	// MySQL packet format: 3-byte length + 1-byte sequence number + payload
	if len(data) < 5 {
		return
	}

	// Read payload length (3 bytes little-endian)
	payloadLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)
	_ = data[3] // sequence number

	if len(data) < 4+payloadLen || payloadLen < 1 {
		return
	}

	commandByte := data[4]
	payload := data[5 : 4+payloadLen]

	switch commandByte {
	case mysqlComQuery:
		query := string(payload)
		h.emitSideEffect(query)
	case mysqlComStmtPrepare:
		query := string(payload)
		h.emitSideEffect(query)
	}
}

func (h *MySQLHook) emitSideEffect(query string) {
	effect := model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     query,
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case h.sideEffects <- effect:
	default:
		logger.Warn("mysql side effect channel full, dropping")
	}
}

// readMySQLPacketLength reads the MySQL packet length (helper function)
func readMySQLPacketLength(data []byte) int {
	if len(data) < 3 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(append(data[:3], 0)))
}

func (h *MySQLHook) registerConn(conn *activeConn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()
	h.activeConns[conn] = struct{}{}
}

func (h *MySQLHook) unregisterConn(conn *activeConn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()
	delete(h.activeConns, conn)
}

func (h *MySQLHook) snapshotActiveConns() []*activeConn {
	h.connsMu.RLock()
	defer h.connsMu.RUnlock()

	conns := make([]*activeConn, 0, len(h.activeConns))
	for conn := range h.activeConns {
		conns = append(conns, conn)
	}
	return conns
}
