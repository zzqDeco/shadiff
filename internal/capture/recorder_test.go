package capture

import (
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func TestNewRecorder_CreatesValidInstance(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	r := NewRecorder("test-session", store)
	if r == nil {
		t.Fatal("expected non-nil recorder")
	}

	// Clean up the background goroutine
	defer r.Stop()

	if r.sessionID != "test-session" {
		t.Fatalf("expected sessionID %q, got %q", "test-session", r.sessionID)
	}
	if r.store != store {
		t.Fatal("expected store to match the provided store")
	}
}

func TestSideEffectChan_ReturnsChannel(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	r := NewRecorder("test-session", store)
	defer r.Stop()

	ch := r.SideEffectChan()
	if ch == nil {
		t.Fatal("expected non-nil side effect channel")
	}

	// Verify the channel is writable by sending a side effect
	se := model.SideEffect{
		Type:   model.SideEffectDB,
		DBType: "mysql",
		Query:  "SELECT 1",
	}

	select {
	case ch <- se:
		// success
	default:
		t.Fatal("expected to be able to send on the side effect channel")
	}
}

func TestCount_ReturnsZeroInitially(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	r := NewRecorder("test-session", store)
	defer r.Stop()

	count := r.Count()
	if count != 0 {
		t.Fatalf("expected initial count 0, got %d", count)
	}
}
