package dbhook

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"shadiff/internal/logger"
	"shadiff/internal/model"
)

type protocolParser interface {
	Feed([]byte) []model.SideEffect
}

type tcpProxy struct {
	dbType      string
	listenAddr  string
	targetAddr  string
	newParser   func() protocolParser
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
	connsMu     sync.RWMutex
	activeConns map[*activeConn]struct{}
}

func newTCPProxy(dbType, listenAddr, targetAddr string, newParser func() protocolParser) *tcpProxy {
	return &tcpProxy{
		dbType:      dbType,
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		newParser:   newParser,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
		activeConns: make(map[*activeConn]struct{}),
	}
}

func (p *tcpProxy) Type() string { return p.dbType }

func (p *tcpProxy) SideEffects() <-chan model.SideEffect {
	return p.sideEffects
}

func (p *tcpProxy) Start(ctx context.Context) error {
	var err error
	p.listener, err = net.Listen("tcp", p.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", p.dbType, "listen", p.listenAddr, "target", p.targetAddr)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-p.done:
				return
			case <-ctx.Done():
				return
			default:
			}

			conn, err := p.listener.Accept()
			if err != nil {
				select {
				case <-p.done:
					return
				case <-ctx.Done():
					return
				default:
					logger.Error(p.dbType+" accept error", err)
					continue
				}
			}

			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				p.handleConn(conn)
			}()
		}
	}()

	return nil
}

func (p *tcpProxy) Flush(ctx context.Context) error {
	for _, conn := range p.snapshotActiveConns() {
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

func (p *tcpProxy) Stop() error {
	close(p.done)
	if p.listener != nil {
		_ = p.listener.Close()
	}
	p.wg.Wait()
	close(p.sideEffects)
	return nil
}

func (p *tcpProxy) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	serverConn, err := net.DialTimeout("tcp", p.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error(p.dbType+" connect target failed", err)
		return
	}
	defer serverConn.Close()

	conn := &activeConn{
		client:  clientConn,
		server:  serverConn,
		flushCh: make(chan chan struct{}),
		done:    make(chan struct{}),
	}
	p.registerConn(conn)
	defer p.unregisterConn(conn)

	parser := p.newParser()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(clientConn, serverConn)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(conn.done)
		p.sniffClientToServer(conn, parser)
	}()

	wg.Wait()
}

func (p *tcpProxy) sniffClientToServer(conn *activeConn, parser protocolParser) {
	buf := make([]byte, 64*1024)
	for {
		select {
		case ack := <-conn.flushCh:
			p.flushConn(conn, buf, parser)
			close(ack)
			continue
		default:
		}

		_ = conn.client.SetReadDeadline(time.Now().Add(hookReadPollInterval))
		n, err := conn.client.Read(buf)
		if n > 0 {
			if _, writeErr := conn.server.Write(buf[:n]); writeErr != nil {
				return
			}
			p.emit(parser.Feed(buf[:n]))
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}
	}
}

func (p *tcpProxy) flushConn(conn *activeConn, buf []byte, parser protocolParser) {
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
			p.emit(parser.Feed(buf[:n]))
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

func (p *tcpProxy) emit(effects []model.SideEffect) {
	for _, effect := range effects {
		select {
		case p.sideEffects <- effect:
		default:
			logger.Warn(p.dbType + " side effect channel full, dropping")
		}
	}
}

func (p *tcpProxy) registerConn(conn *activeConn) {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	p.activeConns[conn] = struct{}{}
}

func (p *tcpProxy) unregisterConn(conn *activeConn) {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	delete(p.activeConns, conn)
}

func (p *tcpProxy) snapshotActiveConns() []*activeConn {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	conns := make([]*activeConn, 0, len(p.activeConns))
	for conn := range p.activeConns {
		conns = append(conns, conn)
	}
	return conns
}
