package dbhook

import (
	"context"
	"encoding/binary"
	"testing"
)

func TestMySQLHook_ParseMySQLPacket_EmitsQuery(t *testing.T) {
	packet := append([]byte{9, 0, 0, 0, mysqlComQuery}, []byte("SELECT 1")...)
	effects := (&mysqlParser{}).Feed(packet)
	if len(effects) != 1 {
		t.Fatalf("effect count = %d, want 1", len(effects))
	}
	if effects[0].SQL().Query != "SELECT 1" {
		t.Fatalf("query = %q, want %q", effects[0].SQL().Query, "SELECT 1")
	}
}

func TestMySQLParser_FragmentedAndPipelinedPackets(t *testing.T) {
	parser := &mysqlParser{}
	first := buildMySQLQueryPacket("SELECT 1")
	second := buildMySQLQueryPacket("SELECT 2")
	wire := append(first, second...)

	if effects := parser.Feed(wire[:3]); len(effects) != 0 {
		t.Fatalf("fragment produced effects = %+v, want none", effects)
	}
	effects := parser.Feed(wire[3:])
	if len(effects) != 2 {
		t.Fatalf("effect count = %d, want 2", len(effects))
	}
	if effects[0].SQL().Query != "SELECT 1" || effects[1].SQL().Query != "SELECT 2" {
		t.Fatalf("queries = %q/%q, want SELECT 1/SELECT 2", effects[0].SQL().Query, effects[1].SQL().Query)
	}
}

func TestPostgresHook_ParsePGMessage_EmitsQuery(t *testing.T) {
	payload := append([]byte("SELECT 1"), 0)
	msg := make([]byte, 1+4+len(payload))
	msg[0] = pgMsgQuery
	binary.BigEndian.PutUint32(msg[1:5], uint32(4+len(payload)))
	copy(msg[5:], payload)

	parser := &postgresParser{startup: false}
	effects := parser.Feed(msg)
	if len(effects) != 1 {
		t.Fatalf("effect count = %d, want 1", len(effects))
	}
	if effects[0].SQL().Query != "SELECT 1" {
		t.Fatalf("query = %q, want %q", effects[0].SQL().Query, "SELECT 1")
	}
}

func TestPostgresParser_FragmentedAndPipelinedMessages(t *testing.T) {
	parser := &postgresParser{startup: true}
	wire := append(buildPGStartupMessage(), buildPGQueryMessage("SELECT 1")...)
	wire = append(wire, buildPGQueryMessage("SELECT 2")...)

	if effects := parser.Feed(wire[:6]); len(effects) != 0 {
		t.Fatalf("fragment produced effects = %+v, want none", effects)
	}
	effects := parser.Feed(wire[6:])
	if len(effects) != 2 {
		t.Fatalf("effect count = %d, want 2", len(effects))
	}
	if effects[0].SQL().Query != "SELECT 1" || effects[1].SQL().Query != "SELECT 2" {
		t.Fatalf("queries = %q/%q, want SELECT 1/SELECT 2", effects[0].SQL().Query, effects[1].SQL().Query)
	}
}

func TestMongoParser_FragmentedAndPipelinedMessages(t *testing.T) {
	parser := &mongoParser{}
	first := buildMongoOpMsg(buildBSONDoc(
		buildBSONString("find", "users"),
		buildBSONString("$db", "testdb"),
	))
	second := buildMongoOpMsg(buildBSONDoc(
		buildBSONString("insert", "orders"),
		buildBSONString("$db", "testdb"),
	))
	wire := append(first, second...)

	if effects := parser.Feed(wire[:10]); len(effects) != 0 {
		t.Fatalf("fragment produced effects = %+v, want none", effects)
	}
	effects := parser.Feed(wire[10:])
	if len(effects) != 2 {
		t.Fatalf("effect count = %d, want 2", len(effects))
	}
	if effects[0].Mongo().Operation != "find" || effects[1].Mongo().Operation != "insert" {
		t.Fatalf("operations = %q/%q, want find/insert", effects[0].Mongo().Operation, effects[1].Mongo().Operation)
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
