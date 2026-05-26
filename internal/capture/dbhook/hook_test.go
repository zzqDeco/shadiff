package dbhook

import (
	"context"
	"errors"
	"testing"
	"time"

	"shadiff/internal/model"
)

type fakeHook struct {
	sideEffects chan model.SideEffect
	flush       func(context.Context) error
}

func (f *fakeHook) Start(ctx context.Context) error { return nil }

func (f *fakeHook) Flush(ctx context.Context) error {
	if f.flush == nil {
		return nil
	}
	return f.flush(ctx)
}

func (f *fakeHook) Stop() error {
	close(f.sideEffects)
	return nil
}

func (f *fakeHook) SideEffects() <-chan model.SideEffect { return f.sideEffects }

func (f *fakeHook) Type() string { return "fake" }

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

func TestNewHook_Redis(t *testing.T) {
	cfg := Config{DBType: "redis", ListenAddr: ":16379", TargetAddr: "127.0.0.1:6379"}
	hook, err := NewHook(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook == nil {
		t.Fatal("expected non-nil hook")
	}
	if _, ok := hook.(*RedisHook); !ok {
		t.Fatalf("expected *RedisHook, got %T", hook)
	}
	if hook.Type() != "redis" {
		t.Fatalf("expected type %q, got %q", "redis", hook.Type())
	}
}

func TestNewHook_Unsupported(t *testing.T) {
	cfg := Config{DBType: "sqlite", ListenAddr: ":15433", TargetAddr: "127.0.0.1:5432"}
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
	if unsupErr.DBType != "sqlite" {
		t.Fatalf("expected DBType %q, got %q", "sqlite", unsupErr.DBType)
	}

	expectedMsg := "unsupported database type: sqlite"
	if err.Error() != expectedMsg {
		t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestGroupFlush_WaitsForAllHooksAndForwarders(t *testing.T) {
	sink := make(chan model.SideEffect, 4)
	first := &fakeHook{sideEffects: make(chan model.SideEffect, 1)}
	first.flush = func(ctx context.Context) error {
		first.sideEffects <- model.SideEffect{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT 1"}
		return nil
	}
	second := &fakeHook{sideEffects: make(chan model.SideEffect, 1)}
	second.flush = func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		second.sideEffects <- model.SideEffect{Type: model.SideEffectDB, DBType: "postgres", Query: "SELECT 2"}
		return nil
	}

	group := NewGroup(context.Background(), []DBHook{first, second}, sink)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := group.Flush(ctx); err != nil {
		t.Fatalf("Flush() error: %v", err)
	}
	if len(sink) != 2 {
		t.Fatalf("len(sink) = %d, want 2", len(sink))
	}

	got := []string{(<-sink).Query, (<-sink).Query}
	if got[0] != "SELECT 1" && got[1] != "SELECT 1" {
		t.Fatalf("missing first query in %+v", got)
	}
	if got[0] != "SELECT 2" && got[1] != "SELECT 2" {
		t.Fatalf("missing second query in %+v", got)
	}
}

func TestSideEffectForwarderWaitDrained_BlocksUntilSinkDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := make(chan model.SideEffect, 1)
	sink := make(chan model.SideEffect)
	forwarder := newSideEffectForwarder(ctx, source, sink)

	source <- model.SideEffect{Type: model.SideEffectDB, DBType: "mysql", Query: "SELECT blocked"}

	deadline := time.After(100 * time.Millisecond)
	for len(source) != 0 {
		select {
		case <-deadline:
			t.Fatal("forwarder did not consume source side effect")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	drainCtx, drainCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer drainCancel()

	if err := forwarder.WaitDrained(drainCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitDrained() error = %v, want context deadline exceeded", err)
	}
}

func TestGroupFlush_RespectsContextTimeout(t *testing.T) {
	hook := &fakeHook{
		sideEffects: make(chan model.SideEffect, 1),
		flush: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	group := NewGroup(context.Background(), []DBHook{hook}, make(chan model.SideEffect, 1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	if err := group.Flush(ctx); err == nil {
		t.Fatal("expected Flush() to return context timeout")
	}
}
