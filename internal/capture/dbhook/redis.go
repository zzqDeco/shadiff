package dbhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"shadiff/internal/logger"
	"shadiff/internal/model"
)

// RedisHook is a Redis protocol proxy that parses client commands.
type RedisHook struct {
	listenAddr  string
	targetAddr  string
	listener    net.Listener
	sideEffects chan model.SideEffect
	done        chan struct{}
	wg          sync.WaitGroup
	connsMu     sync.RWMutex
	activeConns map[*activeConn]struct{}
}

type redisCommand struct {
	Command string
	Key     string
	Args    []string
}

type redisParser struct {
	buf []byte
}

func NewRedisHook(listenAddr, targetAddr string) *RedisHook {
	return &RedisHook{
		listenAddr:  listenAddr,
		targetAddr:  targetAddr,
		sideEffects: make(chan model.SideEffect, 1000),
		done:        make(chan struct{}),
		activeConns: make(map[*activeConn]struct{}),
	}
}

func (h *RedisHook) Type() string { return "redis" }

func (h *RedisHook) SideEffects() <-chan model.SideEffect {
	return h.sideEffects
}

func (h *RedisHook) Start(ctx context.Context) error {
	var err error
	h.listener, err = net.Listen("tcp", h.listenAddr)
	if err != nil {
		return err
	}

	logger.DBHookEvent("started", "redis", "listen", h.listenAddr, "target", h.targetAddr)

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
					logger.Error("redis accept error", err)
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

func (h *RedisHook) Flush(ctx context.Context) error {
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

func (h *RedisHook) Stop() error {
	close(h.done)
	if h.listener != nil {
		h.listener.Close()
	}
	h.wg.Wait()
	close(h.sideEffects)
	return nil
}

func (h *RedisHook) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	serverConn, err := net.DialTimeout("tcp", h.targetAddr, 10*time.Second)
	if err != nil {
		logger.Error("redis connect target failed", err)
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, serverConn)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(conn.done)
		h.sniffClientToServer(conn)
	}()

	wg.Wait()
}

func (h *RedisHook) sniffClientToServer(conn *activeConn) {
	buf := make([]byte, 64*1024)
	parser := &redisParser{}
	for {
		select {
		case ack := <-conn.flushCh:
			h.flushConn(conn, buf, parser)
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

		if _, err := conn.server.Write(buf[:n]); err != nil {
			return
		}
		h.parseRedisData(parser, buf[:n])
	}
}

func (h *RedisHook) flushConn(conn *activeConn, buf []byte, parser *redisParser) {
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
			h.parseRedisData(parser, buf[:n])
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

func (h *RedisHook) parseRedisData(parser *redisParser, data []byte) {
	for _, cmd := range parser.feed(data) {
		h.emitSideEffect(cmd)
	}
}

func (h *RedisHook) emitSideEffect(cmd redisCommand) {
	effect := model.SideEffect{
		Type:         model.SideEffectDB,
		DBType:       "redis",
		RedisCommand: cmd.Command,
		RedisKey:     cmd.Key,
		RedisArgs:    cmd.Args,
		Timestamp:    time.Now().UnixMilli(),
	}

	select {
	case h.sideEffects <- effect:
	default:
		logger.Warn("redis side effect channel full, dropping")
	}
}

func (p *redisParser) feed(data []byte) []redisCommand {
	p.buf = append(p.buf, data...)

	var commands []redisCommand
	for len(p.buf) > 0 {
		parts, consumed, complete, ok := parseRedisCommand(p.buf)
		if !complete {
			break
		}
		if consumed <= 0 {
			p.buf = nil
			break
		}
		if ok {
			if cmd, valid := buildRedisCommand(parts); valid {
				commands = append(commands, cmd)
			}
		}
		p.buf = p.buf[consumed:]
	}

	if len(p.buf) > 1024*1024 {
		p.buf = nil
	}
	return commands
}

func parseRedisCommand(data []byte) ([][]byte, int, bool, bool) {
	if len(data) == 0 {
		return nil, 0, false, true
	}
	if data[0] == '*' {
		return parseRedisArrayCommand(data)
	}
	return parseRedisInlineCommand(data)
}

func parseRedisArrayCommand(data []byte) ([][]byte, int, bool, bool) {
	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd < 0 {
		return nil, 0, false, true
	}

	count, err := strconv.Atoi(string(data[1:lineEnd]))
	if err != nil || count <= 0 {
		return nil, lineEnd + 2, true, false
	}

	pos := lineEnd + 2
	parts := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		if pos >= len(data) {
			return nil, 0, false, true
		}
		if data[pos] != '$' {
			return nil, len(data), true, false
		}

		endRel := bytes.Index(data[pos:], []byte("\r\n"))
		if endRel < 0 {
			return nil, 0, false, true
		}

		bulkLen, err := strconv.Atoi(string(data[pos+1 : pos+endRel]))
		if err != nil || bulkLen < 0 {
			return nil, len(data), true, false
		}

		pos += endRel + 2
		if len(data) < pos+bulkLen+2 {
			return nil, 0, false, true
		}
		if data[pos+bulkLen] != '\r' || data[pos+bulkLen+1] != '\n' {
			return nil, len(data), true, false
		}

		part := append([]byte(nil), data[pos:pos+bulkLen]...)
		parts = append(parts, part)
		pos += bulkLen + 2
	}

	return parts, pos, true, true
}

func parseRedisInlineCommand(data []byte) ([][]byte, int, bool, bool) {
	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd < 0 {
		return nil, 0, false, true
	}

	line := strings.TrimSpace(string(data[:lineEnd]))
	if line == "" {
		return nil, lineEnd + 2, true, false
	}

	fields := strings.Fields(line)
	parts := make([][]byte, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, []byte(field))
	}
	return parts, lineEnd + 2, true, true
}

func buildRedisCommand(parts [][]byte) (redisCommand, bool) {
	if len(parts) == 0 || !utf8.Valid(parts[0]) {
		return redisCommand{}, false
	}

	command := strings.ToUpper(strings.TrimSpace(string(parts[0])))
	if command == "" {
		return redisCommand{}, false
	}

	args := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		args = append(args, encodeRedisArg(part))
	}
	args = redactRedisArgs(command, args)

	return redisCommand{
		Command: command,
		Key:     redisPrimaryKey(command, args),
		Args:    args,
	}, true
}

func encodeRedisArg(arg []byte) string {
	if utf8.Valid(arg) {
		return string(arg)
	}
	return "base64:" + base64.StdEncoding.EncodeToString(arg)
}

func redactRedisArgs(command string, args []string) []string {
	out := append([]string(nil), args...)
	switch command {
	case "AUTH":
		for i := range out {
			out[i] = "<redacted>"
		}
	case "HELLO":
		for i := 0; i < len(out); i++ {
			if strings.EqualFold(out[i], "AUTH") {
				if i+1 < len(out) {
					out[i+1] = "<redacted>"
				}
				if i+2 < len(out) {
					out[i+2] = "<redacted>"
				}
			}
		}
	case "ACL":
		if len(out) >= 2 && strings.EqualFold(out[0], "SETUSER") {
			for i := 2; i < len(out); i++ {
				if isACLSecretToken(out[i]) {
					out[i] = "<redacted>"
				}
			}
		}
	case "CONFIG":
		if len(out) >= 3 && strings.EqualFold(out[0], "SET") && isSensitiveRedisConfigName(out[1]) {
			out[2] = "<redacted>"
		}
	}
	return out
}

func isACLSecretToken(token string) bool {
	return strings.HasPrefix(token, ">") ||
		strings.HasPrefix(token, "<") ||
		strings.HasPrefix(token, "#")
}

func isSensitiveRedisConfigName(name string) bool {
	switch strings.ToLower(name) {
	case "requirepass", "masterauth":
		return true
	default:
		return false
	}
}

func redisPrimaryKey(command string, args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch command {
	case "ACL", "AUTH", "CLIENT", "COMMAND", "CONFIG", "DBSIZE", "DISCARD", "EXEC", "FLUSHALL", "FLUSHDB", "HELLO", "INFO", "MULTI", "PING", "PSUBSCRIBE", "PUBLISH", "PUNSUBSCRIBE", "QUIT", "SELECT", "SUBSCRIBE", "UNSUBSCRIBE", "UNWATCH":
		return ""
	default:
		return args[0]
	}
}

func (h *RedisHook) registerConn(conn *activeConn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()
	h.activeConns[conn] = struct{}{}
}

func (h *RedisHook) unregisterConn(conn *activeConn) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()
	delete(h.activeConns, conn)
}

func (h *RedisHook) snapshotActiveConns() []*activeConn {
	h.connsMu.RLock()
	defer h.connsMu.RUnlock()

	conns := make([]*activeConn, 0, len(h.activeConns))
	for conn := range h.activeConns {
		conns = append(conns, conn)
	}
	return conns
}
