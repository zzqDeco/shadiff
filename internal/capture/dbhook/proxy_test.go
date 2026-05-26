package dbhook

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"shadiff/internal/model"
)

func buildMySQLQueryPacket(query string) []byte {
	payload := append([]byte{mysqlComQuery}, []byte(query)...)
	packet := make([]byte, 4+len(payload))
	packet[0] = byte(len(payload))
	packet[1] = byte(len(payload) >> 8)
	packet[2] = byte(len(payload) >> 16)
	packet[3] = 0
	copy(packet[4:], payload)
	return packet
}

func buildPGStartupMessage() []byte {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint32(msg[0:4], 8)
	binary.BigEndian.PutUint32(msg[4:8], 196608)
	return msg
}

func buildPGQueryMessage(query string) []byte {
	payload := append([]byte(query), 0)
	msg := make([]byte, 1+4+len(payload))
	msg[0] = pgMsgQuery
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)
	return msg
}

func buildMongoOpMsg(doc []byte) []byte {
	body := make([]byte, 4, 5+len(doc))
	body = append(body, 0)
	body = append(body, doc...)

	header := make([]byte, 16)
	binary.LittleEndian.PutUint32(header[0:4], uint32(16+len(body)))
	binary.LittleEndian.PutUint32(header[12:16], uint32(opMsgOpCode))
	return append(header, body...)
}

func buildRedisCommandWire(parts ...string) []byte {
	rawParts := make([][]byte, 0, len(parts))
	for _, part := range parts {
		rawParts = append(rawParts, []byte(part))
	}
	return buildRedisArray(rawParts...)
}

func waitForSideEffect(t *testing.T, ch <-chan model.SideEffect) model.SideEffect {
	t.Helper()

	select {
	case effect := <-ch:
		return effect
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for side effect")
		return model.SideEffect{}
	}
}

func startTargetServer(t *testing.T, handler func(net.Conn) error) (string, <-chan error) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()

		done <- handler(conn)
	}()

	return listener.Addr().String(), done
}

func waitForHandleConn(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handleConn to exit")
	}
}

func TestReadMySQLPacketLength(t *testing.T) {
	packet := []byte{0x09, 0x01, 0x00}

	if got := readMySQLPacketLength(packet); got != 265 {
		t.Fatalf("readMySQLPacketLength() = %d, want 265", got)
	}
	if got := readMySQLPacketLength([]byte{0x01, 0x02}); got != 0 {
		t.Fatalf("short packet length = %d, want 0", got)
	}
}

func TestNullTermHelpers(t *testing.T) {
	if got := extractNullTermString([]byte("hello\x00world")); got != "hello" {
		t.Fatalf("extractNullTermString() = %q, want %q", got, "hello")
	}
	if got := nullTermIndex([]byte("abc\x00def")); got != 3 {
		t.Fatalf("nullTermIndex() = %d, want 3", got)
	}
	if got := nullTermIndex([]byte("abcdef")); got != -1 {
		t.Fatalf("nullTermIndex() without terminator = %d, want -1", got)
	}
}

func TestMySQLHook_HandleConn_ForwardsTrafficAndCapturesQuery(t *testing.T) {
	packet := buildMySQLQueryPacket("SELECT 1")
	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		buf := make([]byte, len(packet))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		if !bytes.Equal(buf, packet) {
			return fmt.Errorf("forwarded packet = %v, want %v", buf, packet)
		}
		_, err := conn.Write([]byte("OK"))
		return err
	})

	hook := NewMySQLHook(":0", targetAddr)
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		hook.handleConn(proxyConn)
	}()

	if _, err := clientConn.Write(packet); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error: %v", err)
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("ReadFull() error: %v", err)
	}
	if string(resp) != "OK" {
		t.Fatalf("response = %q, want %q", string(resp), "OK")
	}
	_ = clientConn.Close()

	if err := <-serverDone; err != nil {
		t.Fatalf("target server error: %v", err)
	}
	waitForHandleConn(t, handleDone)

	effect := waitForSideEffect(t, hook.SideEffects())
	if effect.Query != "SELECT 1" {
		t.Fatalf("query = %q, want %q", effect.Query, "SELECT 1")
	}
}

func TestPostgresHook_HandleConn_ForwardsTrafficAndCapturesQuery(t *testing.T) {
	startup := buildPGStartupMessage()
	query := buildPGQueryMessage("SELECT 1")

	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		startupBuf := make([]byte, len(startup))
		if _, err := io.ReadFull(conn, startupBuf); err != nil {
			return err
		}
		if !bytes.Equal(startupBuf, startup) {
			return fmt.Errorf("startup message = %v, want %v", startupBuf, startup)
		}

		queryBuf := make([]byte, len(query))
		if _, err := io.ReadFull(conn, queryBuf); err != nil {
			return err
		}
		if !bytes.Equal(queryBuf, query) {
			return fmt.Errorf("query message = %v, want %v", queryBuf, query)
		}

		_, err := conn.Write([]byte("R"))
		return err
	})

	hook := NewPostgresHook(":0", targetAddr)
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		hook.handleConn(proxyConn)
	}()

	if _, err := clientConn.Write(startup); err != nil {
		t.Fatalf("Write(startup) error: %v", err)
	}
	if _, err := clientConn.Write(query); err != nil {
		t.Fatalf("Write(query) error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error: %v", err)
	}
	resp := make([]byte, 1)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("ReadFull() error: %v", err)
	}
	if string(resp) != "R" {
		t.Fatalf("response = %q, want %q", string(resp), "R")
	}
	_ = clientConn.Close()

	if err := <-serverDone; err != nil {
		t.Fatalf("target server error: %v", err)
	}
	waitForHandleConn(t, handleDone)

	effect := waitForSideEffect(t, hook.SideEffects())
	if effect.Query != "SELECT 1" {
		t.Fatalf("query = %q, want %q", effect.Query, "SELECT 1")
	}
}

func TestMongoHook_HandleConn_ForwardsTrafficAndCapturesCommand(t *testing.T) {
	message := buildMongoOpMsg(buildBSONDoc(
		buildBSONString("find", "users"),
		buildBSONString("$db", "testdb"),
	))

	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		buf := make([]byte, len(message))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		if !bytes.Equal(buf, message) {
			return fmt.Errorf("forwarded message = %v, want %v", buf, message)
		}

		_, err := conn.Write([]byte("M"))
		return err
	})

	hook := NewMongoHook(":0", targetAddr)
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		hook.handleConn(proxyConn)
	}()

	if _, err := clientConn.Write(message); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error: %v", err)
	}
	resp := make([]byte, 1)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("ReadFull() error: %v", err)
	}
	if string(resp) != "M" {
		t.Fatalf("response = %q, want %q", string(resp), "M")
	}
	_ = clientConn.Close()

	if err := <-serverDone; err != nil {
		t.Fatalf("target server error: %v", err)
	}
	waitForHandleConn(t, handleDone)

	effect := waitForSideEffect(t, hook.SideEffects())
	if effect.Operation != "find" {
		t.Fatalf("operation = %q, want %q", effect.Operation, "find")
	}
	if effect.Collection != "users" {
		t.Fatalf("collection = %q, want %q", effect.Collection, "users")
	}
	if effect.Database != "testdb" {
		t.Fatalf("database = %q, want %q", effect.Database, "testdb")
	}
}

func TestRedisHook_HandleConn_ForwardsTrafficAndCapturesCommand(t *testing.T) {
	message := buildRedisCommandWire("SET", "user:1", "ada")
	targetAddr, serverDone := startTargetServer(t, func(conn net.Conn) error {
		buf := make([]byte, len(message))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		if !bytes.Equal(buf, message) {
			return fmt.Errorf("forwarded message = %v, want %v", buf, message)
		}

		_, err := conn.Write([]byte("+OK\r\n"))
		return err
	})

	hook := NewRedisHook(":0", targetAddr)
	proxyConn, clientConn := net.Pipe()
	defer clientConn.Close()

	handleDone := make(chan struct{})
	go func() {
		defer close(handleDone)
		hook.handleConn(proxyConn)
	}()

	if _, err := clientConn.Write(message); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error: %v", err)
	}
	resp := make([]byte, 5)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("ReadFull() error: %v", err)
	}
	if string(resp) != "+OK\r\n" {
		t.Fatalf("response = %q, want %q", string(resp), "+OK\\r\\n")
	}
	_ = clientConn.Close()

	if err := <-serverDone; err != nil {
		t.Fatalf("target server error: %v", err)
	}
	waitForHandleConn(t, handleDone)

	effect := waitForSideEffect(t, hook.SideEffects())
	if effect.RedisCommand != "SET" {
		t.Fatalf("redis command = %q, want SET", effect.RedisCommand)
	}
	if effect.RedisKey != "user:1" {
		t.Fatalf("redis key = %q, want user:1", effect.RedisKey)
	}
	if len(effect.RedisArgs) != 2 || effect.RedisArgs[0] != "user:1" || effect.RedisArgs[1] != "ada" {
		t.Fatalf("redis args = %+v, want [user:1 ada]", effect.RedisArgs)
	}
}

func TestPostgresHook_StartStopLifecycle(t *testing.T) {
	hook := NewPostgresHook("127.0.0.1:0", "127.0.0.1:5432")
	if err := hook.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := hook.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestMongoHook_StartStopLifecycle(t *testing.T) {
	hook := NewMongoHook("127.0.0.1:0", "127.0.0.1:27017")
	if err := hook.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := hook.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestRedisHook_StartStopLifecycle(t *testing.T) {
	hook := NewRedisHook("127.0.0.1:0", "127.0.0.1:6379")
	if err := hook.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := hook.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}
