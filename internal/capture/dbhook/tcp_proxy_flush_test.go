package dbhook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"shadiff/internal/model"
)

type testProtocolParser struct {
	emit bool
}

func (p *testProtocolParser) Feed(data []byte) []model.SideEffect {
	if !p.emit {
		return nil
	}
	return []model.SideEffect{
		model.NewSQLSideEffect("mysql", string(data), time.Now().UnixMilli()),
	}
}

func TestTCPProxyFlush_ForwardsClientTrafficAndEmitsSideEffect(t *testing.T) {
	message := []byte("SELECT 1")
	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		got := make([]byte, len(message))
		if _, err := io.ReadFull(conn, got); err != nil {
			return err
		}
		if string(got) != string(message) {
			return fmt.Errorf("forwarded message = %q, want %q", string(got), string(message))
		}
		return nil
	})

	proxy := newTCPProxy("test", "127.0.0.1:0", targetAddr, func() protocolParser {
		return &testProtocolParser{emit: true}
	})
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		proxy.handleConn(proxyConn)
	}()
	waitForActiveTCPProxyConn(t, proxy)

	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write(message)
		writeDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := proxy.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("client Write() error = %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("target server error = %v", err)
	}

	effect := waitForSideEffect(t, proxy.SideEffects())
	if effect.SQL().Query != string(message) {
		t.Fatalf("side effect query = %q, want %q", effect.SQL().Query, string(message))
	}

	_ = clientConn.Close()
	waitForHandleConn(t, handleDone)
}

func TestTCPProxyFlush_RespectsContextTimeoutWhenConnectionCannotAck(t *testing.T) {
	proxy := newTCPProxy("test", "127.0.0.1:0", "127.0.0.1:1", func() protocolParser {
		return &testProtocolParser{emit: true}
	})
	conn := &activeConn{
		flushCh: make(chan chan struct{}),
		done:    make(chan struct{}),
	}
	proxy.registerConn(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := proxy.Flush(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Flush() error = %v, want context deadline exceeded", err)
	}
}

func TestTCPProxyFlush_SkipsClosedConnections(t *testing.T) {
	proxy := newTCPProxy("test", "127.0.0.1:0", "127.0.0.1:1", func() protocolParser {
		return &testProtocolParser{emit: true}
	})
	closed := make(chan struct{})
	close(closed)
	conn := &activeConn{
		flushCh: make(chan chan struct{}),
		done:    closed,
	}
	proxy.registerConn(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := proxy.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v, want nil", err)
	}
}

func TestTCPProxyFlush_ForwardsMalformedTrafficWithoutSideEffect(t *testing.T) {
	message := []byte("not-a-parseable-command")
	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		got := make([]byte, len(message))
		if _, err := io.ReadFull(conn, got); err != nil {
			return err
		}
		if string(got) != string(message) {
			return fmt.Errorf("forwarded message = %q, want %q", string(got), string(message))
		}
		return nil
	})

	proxy := newTCPProxy("test", "127.0.0.1:0", targetAddr, func() protocolParser {
		return &testProtocolParser{emit: false}
	})
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		proxy.handleConn(proxyConn)
	}()
	waitForActiveTCPProxyConn(t, proxy)

	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write(message)
		writeDone <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := proxy.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("client Write() error = %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("target server error = %v", err)
	}

	select {
	case effect := <-proxy.SideEffects():
		t.Fatalf("unexpected side effect: %+v", effect)
	default:
	}

	_ = clientConn.Close()
	waitForHandleConn(t, handleDone)
}

func waitForActiveTCPProxyConn(t *testing.T, proxy *tcpProxy) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		if len(proxy.snapshotActiveConns()) > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for active proxy connection")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
