package dbhook

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"shadiff/internal/dbtype"
	"shadiff/internal/model"
)

// RedisHook is a Redis protocol proxy that parses client commands.
type RedisHook struct {
	*tcpProxy
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
		tcpProxy: newTCPProxy(dbtype.Redis, listenAddr, targetAddr, func() protocolParser {
			return &redisParser{}
		}),
	}
}

func (p *redisParser) Feed(data []byte) []model.SideEffect {
	commands := p.feed(data)
	effects := make([]model.SideEffect, 0, len(commands))
	for _, cmd := range commands {
		effects = append(effects, model.NewRedisSideEffect(cmd.Command, cmd.Key, cmd.Args, time.Now().UnixMilli()))
	}
	return effects
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
