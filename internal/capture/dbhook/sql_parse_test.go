package dbhook

import (
	"context"
	"encoding/binary"
	"testing"
)

func TestMySQLHook_ParseMySQLPacket_EmitsQuery(t *testing.T) {
	hook := NewMySQLHook(":0", "127.0.0.1:3306")
	packet := append([]byte{9, 0, 0, 0, mysqlComQuery}, []byte("SELECT 1")...)

	hook.parseMySQLPacket(packet)

	select {
	case effect := <-hook.SideEffects():
		if effect.Query != "SELECT 1" {
			t.Fatalf("query = %q, want %q", effect.Query, "SELECT 1")
		}
	default:
		t.Fatal("expected MySQL query side effect")
	}
}

func TestPostgresHook_ParsePGMessage_EmitsQuery(t *testing.T) {
	hook := NewPostgresHook(":0", "127.0.0.1:5432")
	payload := append([]byte("SELECT 1"), 0)
	msg := make([]byte, 1+4+len(payload))
	msg[0] = pgMsgQuery
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)

	hook.parsePGMessage(msg)

	select {
	case effect := <-hook.SideEffects():
		if effect.Query != "SELECT 1" {
			t.Fatalf("query = %q, want %q", effect.Query, "SELECT 1")
		}
	default:
		t.Fatal("expected Postgres query side effect")
	}
}

func TestMySQLHook_StartStopLifecycle(t *testing.T) {
	hook := NewMySQLHook("127.0.0.1:0", "127.0.0.1:3306")
	if err := hook.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if err := hook.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}
