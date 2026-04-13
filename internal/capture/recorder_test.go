package capture

import (
	"testing"

	"shadiff/internal/model"
	"shadiff/internal/storage"
)

func newRecorderTestFixture(t *testing.T) (*storage.FileStore, *model.Session, *Recorder) {
	t.Helper()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	session := &model.Session{
		Name:   "recorder-test",
		Status: model.SessionRecording,
		Source: model.EndpointConfig{BaseURL: "http://source"},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	recorder := NewRecorder(session.ID, store)
	return store, session, recorder
}

func TestNewRecorder_CreatesValidInstance(t *testing.T) {
	store, session, r := newRecorderTestFixture(t)
	if r == nil {
		t.Fatal("expected non-nil recorder")
	}

	// Clean up the background goroutine
	defer r.Stop()

	if r.sessionID != session.ID {
		t.Fatalf("expected sessionID %q, got %q", session.ID, r.sessionID)
	}
	if r.store != store {
		t.Fatal("expected store to match the provided store")
	}
}

func TestSideEffectChan_ReturnsChannel(t *testing.T) {
	_, _, r := newRecorderTestFixture(t)
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
	_, _, r := newRecorderTestFixture(t)
	defer r.Stop()

	count := r.Count()
	if count != 0 {
		t.Fatalf("expected initial count 0, got %d", count)
	}
}

func TestFinishRequestScope_AttachesAttributedSideEffects(t *testing.T) {
	store, session, r := newRecorderTestFixture(t)
	defer r.Stop()

	scopeID := r.BeginRequestScope(100)
	r.SideEffectChan() <- model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     "SELECT 1",
		Timestamp: 105,
	}

	record := &model.Record{
		ID:         "rec-1",
		Sequence:   1,
		RecordedAt: 110,
	}
	if err := r.FinishRequestScope(scopeID, record); err != nil {
		t.Fatalf("FinishRequestScope() error: %v", err)
	}

	if len(record.SideEffects) != 1 {
		t.Fatalf("sideEffects len = %d, want 1", len(record.SideEffects))
	}
	if record.SideEffects[0].Query != "SELECT 1" {
		t.Fatalf("side effect query = %q, want %q", record.SideEffects[0].Query, "SELECT 1")
	}

	records, err := store.ListRecords(session.ID)
	if err != nil {
		t.Fatalf("ListRecords() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	if len(records[0].SideEffects) != 1 {
		t.Fatalf("persisted sideEffects len = %d, want 1", len(records[0].SideEffects))
	}
}

func TestFinishRequestScope_OverlappingRequestsUseLatestMatchingScope(t *testing.T) {
	_, _, r := newRecorderTestFixture(t)
	defer r.Stop()

	firstScope := r.BeginRequestScope(100)
	secondScope := r.BeginRequestScope(105)

	r.SideEffectChan() <- model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     "SELECT before second",
		Timestamp: 104,
	}
	r.SideEffectChan() <- model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     "SELECT after second",
		Timestamp: 106,
	}

	firstRecord := &model.Record{ID: "rec-1", Sequence: 1, RecordedAt: 120}
	secondRecord := &model.Record{ID: "rec-2", Sequence: 2, RecordedAt: 115}

	if err := r.FinishRequestScope(secondScope, secondRecord); err != nil {
		t.Fatalf("FinishRequestScope(second) error: %v", err)
	}
	if err := r.FinishRequestScope(firstScope, firstRecord); err != nil {
		t.Fatalf("FinishRequestScope(first) error: %v", err)
	}

	if len(firstRecord.SideEffects) != 1 || firstRecord.SideEffects[0].Query != "SELECT before second" {
		t.Fatalf("first record sideEffects = %#v, want first query only", firstRecord.SideEffects)
	}
	if len(secondRecord.SideEffects) != 1 || secondRecord.SideEffects[0].Query != "SELECT after second" {
		t.Fatalf("second record sideEffects = %#v, want second query only", secondRecord.SideEffects)
	}
}

func TestFinishRequestScope_DropsOrphanSideEffects(t *testing.T) {
	_, _, r := newRecorderTestFixture(t)
	defer r.Stop()

	firstScope := r.BeginRequestScope(100)
	firstRecord := &model.Record{ID: "rec-1", Sequence: 1, RecordedAt: 110}
	if err := r.FinishRequestScope(firstScope, firstRecord); err != nil {
		t.Fatalf("FinishRequestScope(first) error: %v", err)
	}

	r.SideEffectChan() <- model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "mysql",
		Query:     "SELECT orphan",
		Timestamp: 111,
	}

	secondScope := r.BeginRequestScope(120)
	secondRecord := &model.Record{ID: "rec-2", Sequence: 2, RecordedAt: 130}
	if err := r.FinishRequestScope(secondScope, secondRecord); err != nil {
		t.Fatalf("FinishRequestScope(second) error: %v", err)
	}

	if len(secondRecord.SideEffects) != 0 {
		t.Fatalf("second record sideEffects len = %d, want 0", len(secondRecord.SideEffects))
	}
}

func TestStop_DrainsSideEffectsBeforeReturning(t *testing.T) {
	_, _, r := newRecorderTestFixture(t)

	scopeID := r.BeginRequestScope(100)
	r.SideEffectChan() <- model.SideEffect{
		Type:      model.SideEffectDB,
		DBType:    "postgres",
		Query:     "SELECT stop drain",
		Timestamp: 105,
	}

	r.Stop()

	record := &model.Record{ID: "rec-stop", Sequence: 1, RecordedAt: 110}
	if err := r.FinishRequestScope(scopeID, record); err != nil {
		t.Fatalf("FinishRequestScope() after Stop error: %v", err)
	}

	if len(record.SideEffects) != 1 {
		t.Fatalf("sideEffects len = %d, want 1", len(record.SideEffects))
	}
}
