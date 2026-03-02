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

// PostgresHook is a PostgreSQL protocol proxy that parses Simple/Extended Query messages
type PostgresHook struct {
	listenAddr  string
	targetAddr  string
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
}

// PostgreSQL frontend message types
const (
	pgMsgQuery = 'Q' // Simple Query
	pgMsgParse = 'P' // Extended Query: Parse
)

func NewPostgresHook(listenAddr, targetAddr string) *PostgresHook {
	return &PostgresHook{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
	}
}

func (h *PostgresHook) Type() string { return "postgres" }

func (h *PostgresHook) SideEffects() <-chan model.SideEffect {
	return h.sideEffects
}

func (h *PostgresHook) Start(ctx context.Context) error {
	var err error
	h.listener, err = net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", "postgres", "listen", h.listenAddr, "target", h.targetAddr)

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
					logger.Error("postgres accept error", err)
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

func (h *PostgresHook) Stop() error {
	close(h.done)
	if h.listener != nil {
		h.listener.Close()
	}
	h.wg.Wait()
	close(h.sideEffects)
	return nil
}

func (h *PostgresHook) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	serverConn, err := net.DialTimeout("tcp", h.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("postgres connect target failed", err)
		return
	}
	defer serverConn.Close()

	var wg sync.WaitGroup

	// Server -> Client (passthrough)
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()

	// Client -> Server (sniff)
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.sniffClientToServer(clientConn, serverConn)
	}()

	wg.Wait()
}

func (h *PostgresHook) sniffClientToServer(client, server net.Conn) {
	buf := make([]byte, 64*1024)
	// Flag whether the startup phase has passed (startup messages have no type byte)
	startup := true

	for {
		n, err := client.Read(buf)
		if err != nil {
			return
		}

		// Forward
		if _, err := server.Write(buf[:n]); err != nil {
			return
		}

		if startup {
			// Startup message format: 4-byte length + 4-byte protocol version + ...
			// Skip startup phase messages
			if n >= 8 {
				startup = false
			}
			continue
		}

		// Parse PostgreSQL frontend messages
		h.parsePGMessage(buf[:n])
	}
}

// parsePGMessage parses PostgreSQL frontend messages
func (h *PostgresHook) parsePGMessage(data []byte) {
	offset := 0
	for offset < len(data) {
		if offset+5 > len(data) {
			break
		}

		msgType := data[offset]
		msgLen := int(binary.BigEndian.Uint32(data[offset+1 : offset+5]))

		if msgLen < 4 || offset+1+msgLen > len(data) {
			break
		}

		payload := data[offset+5 : offset+1+msgLen]

		switch msgType {
		case pgMsgQuery:
			// Simple Query: null-terminated string
			query := extractNullTermString(payload)
			if query != "" {
				h.emitSideEffect(query)
			}
		case pgMsgParse:
			// Parse: stmt_name(null) + query(null) + ...
			// Skip statement name
			idx := nullTermIndex(payload)
			if idx >= 0 && idx+1 < len(payload) {
				query := extractNullTermString(payload[idx+1:])
				if query != "" {
					h.emitSideEffect(query)
				}
			}
		}

		offset += 1 + msgLen
	}
}

func (h *PostgresHook) emitSideEffect(query string) {
	effect := model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "postgres",
		Query:     query,
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case h.sideEffects <- effect:
	default:
		logger.Warn("postgres side effect channel full, dropping")
	}
}

func extractNullTermString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

func nullTermIndex(data []byte) int {
	for i, b := range data {
		if b == 0 {
			return i
		}
	}
	return -1
}
