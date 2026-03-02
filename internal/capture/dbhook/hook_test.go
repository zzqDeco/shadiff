package dbhook

import (
	"errors"
	"testing"
)

func TestNewHook_MySQL(t *testing.T) {
	cfg := Config{DBType: "mysql", ListenAddr: ":13306", TargetAddr: "127.0.0.1:3306"}
	hook, err := NewHook(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if _, ok := hook.(*MySQLHook); !ok {
		t.Fatalf("expected *MySQLHook, got %T", hook)
	}
	if hook.Type() != "mysql" {
		t.Fatalf("expected type %q, got %q", "mysql", hook.Type())
	}
}

func TestNewHook_Postgres(t *testing.T) {
	cfg := Config{DBType: "postgres", ListenAddr: ":15432", TargetAddr: "127.0.0.1:5432"}
	hook, err := NewHook(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if _, ok := hook.(*PostgresHook); !ok {
		t.Fatalf("expected *PostgresHook, got %T", hook)
	}
	if hook.Type() != "postgres" {
		t.Fatalf("expected type %q, got %q", "postgres", hook.Type())
	}
}

func TestNewHook_Mongo(t *testing.T) {
	cfg := Config{DBType: "mongo", ListenAddr: ":17017", TargetAddr: "127.0.0.1:27017"}
	hook, err := NewHook(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if _, ok := hook.(*MongoHook); !ok {
		t.Fatalf("expected *MongoHook, got %T", hook)
	}
	if hook.Type() != "mongo" {
		t.Fatalf("expected type %q, got %q", "mongo", hook.Type())
	}
}

func TestNewHook_Unsupported(t *testing.T) {
	cfg := Config{DBType: "redis", ListenAddr: ":16379", TargetAddr: "127.0.0.1:6379"}
	hook, err := NewHook(cfg)
	if hook != nil {
		t.Fatal("expected nil hook for unsupported type")
	}
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}

	var unsupErr *UnsupportedDBError
	if !errors.As(err, &unsupErr) {
		t.Fatalf("expected *UnsupportedDBError, got %T", err)
	}
	if unsupErr.DBType != "redis" {
		t.Fatalf("expected DBType %q, got %q", "redis", unsupErr.DBType)
	}

	expectedMsg := "unsupported database type: redis"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}
