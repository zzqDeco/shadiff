package dbhook

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func buildRedisArray(parts ...[]byte) []byte {
	out := []byte("*" + itoa(len(parts)) + "\r\n")
	for _, part := range parts {
		out = append(out, []byte("$"+itoa(len(part))+"\r\n")...)
		out = append(out, part...)
		out = append(out, '\r', '\n')
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestRedisParser_RESPCommands(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		want redisCommand
	}{
		{
			name: "get",
			wire: buildRedisArray([]byte("get"), []byte("user:1")),
			want: redisCommand{Command: "GET", Key: "user:1", Args: []string{"user:1"}},
		},
		{
			name: "set",
			wire: buildRedisArray([]byte("SET"), []byte("user:1"), []byte("ada")),
			want: redisCommand{Command: "SET", Key: "user:1", Args: []string{"user:1", "ada"}},
		},
		{
			name: "hset",
			wire: buildRedisArray([]byte("HSET"), []byte("user:1"), []byte("name"), []byte("ada")),
			want: redisCommand{Command: "HSET", Key: "user:1", Args: []string{"user:1", "name", "ada"}},
		},
		{
			name: "del",
			wire: buildRedisArray([]byte("DEL"), []byte("user:1")),
			want: redisCommand{Command: "DEL", Key: "user:1", Args: []string{"user:1"}},
		},
		{
			name: "ping",
			wire: buildRedisArray([]byte("PING")),
			want: redisCommand{Command: "PING", Args: []string{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &redisParser{}
			got := parser.feed(tt.wire)
			if len(got) != 1 {
				t.Fatalf("parsed command count = %d, want 1: %+v", len(got), got)
			}
			if got[0].Command != tt.want.Command || got[0].Key != tt.want.Key || !reflect.DeepEqual(got[0].Args, tt.want.Args) {
				if len(got[0].Args) == 0 && len(tt.want.Args) == 0 && got[0].Command == tt.want.Command && got[0].Key == tt.want.Key {
					return
				}
				t.Fatalf("parsed command = %+v, want %+v", got[0], tt.want)
			}
		})
	}
}

func TestRedisParser_InlineCommand(t *testing.T) {
	parser := &redisParser{}
	got := parser.feed([]byte("get user:1\r\n"))
	if len(got) != 1 {
		t.Fatalf("parsed command count = %d, want 1", len(got))
	}
	if got[0].Command != "GET" || got[0].Key != "user:1" {
		t.Fatalf("parsed command = %+v", got[0])
	}
}

func TestRedisParser_PipelinedCommands(t *testing.T) {
	wire := append(buildRedisArray([]byte("SET"), []byte("k1"), []byte("v1")), buildRedisArray([]byte("GET"), []byte("k1"))...)

	parser := &redisParser{}
	got := parser.feed(wire)
	if len(got) != 2 {
		t.Fatalf("parsed command count = %d, want 2: %+v", len(got), got)
	}
	if got[0].Command != "SET" || got[1].Command != "GET" {
		t.Fatalf("parsed commands = %+v", got)
	}
}

func TestRedisParser_FragmentedCommand(t *testing.T) {
	wire := buildRedisArray([]byte("SET"), []byte("user:1"), []byte("ada"))
	split := len(wire) / 2

	parser := &redisParser{}
	if got := parser.feed(wire[:split]); len(got) != 0 {
		t.Fatalf("fragment produced commands: %+v", got)
	}
	got := parser.feed(wire[split:])
	if len(got) != 1 {
		t.Fatalf("parsed command count = %d, want 1", len(got))
	}
	if got[0].Command != "SET" || got[0].Key != "user:1" {
		t.Fatalf("parsed command = %+v", got[0])
	}
}

func TestRedisParser_MalformedInputProducesNoSideEffect(t *testing.T) {
	parser := &redisParser{}
	if got := parser.feed([]byte("*2\r\n!bad\r\n")); len(got) != 0 {
		t.Fatalf("malformed input produced commands: %+v", got)
	}
}

func TestRedisParser_EncodesNonUTF8Args(t *testing.T) {
	parser := &redisParser{}
	got := parser.feed(buildRedisArray([]byte("SET"), []byte{0xff, 0x00}, []byte("value")))
	if len(got) != 1 {
		t.Fatalf("parsed command count = %d, want 1", len(got))
	}
	wantKey := "base64:" + base64.StdEncoding.EncodeToString([]byte{0xff, 0x00})
	if got[0].Key != wantKey || got[0].Args[0] != wantKey {
		t.Fatalf("binary key = %q args=%+v, want %q", got[0].Key, got[0].Args, wantKey)
	}
}

func TestRedisParser_RedactsSensitiveArgs(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		args []string
	}{
		{
			name: "auth",
			wire: buildRedisArray([]byte("AUTH"), []byte("secret")),
			args: []string{"<redacted>"},
		},
		{
			name: "hello auth",
			wire: buildRedisArray([]byte("HELLO"), []byte("3"), []byte("AUTH"), []byte("default"), []byte("secret")),
			args: []string{"3", "AUTH", "<redacted>", "<redacted>"},
		},
		{
			name: "acl setuser",
			wire: buildRedisArray([]byte("ACL"), []byte("SETUSER"), []byte("default"), []byte(">secret"), []byte("~*")),
			args: []string{"SETUSER", "default", "<redacted>", "~*"},
		},
		{
			name: "config requirepass",
			wire: buildRedisArray([]byte("CONFIG"), []byte("SET"), []byte("requirepass"), []byte("secret")),
			args: []string{"SET", "requirepass", "<redacted>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &redisParser{}
			got := parser.feed(tt.wire)
			if len(got) != 1 {
				t.Fatalf("parsed command count = %d, want 1", len(got))
			}
			if !reflect.DeepEqual(got[0].Args, tt.args) {
				t.Fatalf("args = %+v, want %+v", got[0].Args, tt.args)
			}
		})
	}
}
