package cmd

import (
	"context"
	"errors"
	"testing"

	"shadiff/internal/capture/dbhook"
	"shadiff/internal/config"
	"shadiff/internal/model"
)

type fakeDBHook struct {
	startErr     error
	flushErr     error
	stopped      bool
	sideEffects  chan model.SideEffect
	startInvoked bool
	flushInvoked bool
}

func (f *fakeDBHook) Start(ctx context.Context) error {
	f.startInvoked = true
	return f.startErr
}

func (f *fakeDBHook) Stop() error {
	f.stopped = true
	if f.sideEffects != nil {
		close(f.sideEffects)
	}
	return nil
}

func (f *fakeDBHook) Flush(_ context.Context) error {
	f.flushInvoked = true
	return f.flushErr
}

func (f *fakeDBHook) SideEffects() <-chan model.SideEffect {
	return f.sideEffects
}

func (f *fakeDBHook) Type() string { return "fake" }

func TestParseDBProxySpec(t *testing.T) {
	proxy, err := parseDBProxySpec("mysql://:13306->127.0.0.1:3306")
	if err != nil {
		t.Fatalf("parseDBProxySpec() error: %v", err)
	}
	if proxy.Type != "mysql" || proxy.ListenAddr != ":13306" || proxy.TargetAddr != "127.0.0.1:3306" {
		t.Fatalf("unexpected proxy: %+v", proxy)
	}
}

func TestParseDBProxySpec_Invalid(t *testing.T) {
	if _, err := parseDBProxySpec("not-valid"); err == nil {
		t.Fatal("expected invalid db proxy to return an error")
	}
}

func TestResolveRecordDBProxies_UsesConfigWhenFlagUnchanged(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Capture.DBProxies = []config.DBProxyConfig{
		{Type: "postgres", ListenAddr: ":15432", TargetAddr: "127.0.0.1:5432"},
	}

	proxies, err := resolveRecordDBProxies(false, nil, cfg)
	if err != nil {
		t.Fatalf("resolveRecordDBProxies() error: %v", err)
	}
	if len(proxies) != 1 || proxies[0].Type != "postgres" {
		t.Fatalf("unexpected proxies: %+v", proxies)
	}
}

func TestResolveReplayDBProxies_UsesConfigWhenFlagUnchanged(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Replay.DBProxies = []config.DBProxyConfig{
		{Type: "postgres", ListenAddr: ":15433", TargetAddr: "127.0.0.1:5432"},
	}

	proxies, err := resolveReplayDBProxies(false, nil, cfg)
	if err != nil {
		t.Fatalf("resolveReplayDBProxies() error: %v", err)
	}
	if len(proxies) != 1 || proxies[0].Type != "postgres" {
		t.Fatalf("unexpected proxies: %+v", proxies)
	}
}

func TestResolveReplayDBProxies_FlagsOverrideConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Replay.DBProxies = []config.DBProxyConfig{
		{Type: "postgres", ListenAddr: ":15433", TargetAddr: "127.0.0.1:5432"},
	}

	proxies, err := resolveReplayDBProxies(true, []string{"mysql://:13307->127.0.0.1:3306"}, cfg)
	if err != nil {
		t.Fatalf("resolveReplayDBProxies() error: %v", err)
	}
	if len(proxies) != 1 || proxies[0].Type != "mysql" {
		t.Fatalf("unexpected proxies: %+v", proxies)
	}
}

func TestStartDBHooks_StopsStartedHooksOnError(t *testing.T) {
	first := &fakeDBHook{sideEffects: make(chan model.SideEffect)}
	expectedErr := errors.New("boom")

	oldFactory := newDBHook
	defer func() { newDBHook = oldFactory }()

	callCount := 0
	newDBHook = func(cfg dbhook.Config) (dbhook.DBHook, error) {
		callCount++
		if callCount == 1 {
			return first, nil
		}
		return nil, expectedErr
	}

	sink := make(chan model.SideEffect, 1)

	_, err := startDBHooks(context.Background(), sink, []config.DBProxyConfig{
		{Type: "mysql", ListenAddr: ":1", TargetAddr: ":2"},
		{Type: "postgres", ListenAddr: ":3", TargetAddr: ":4"},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("startDBHooks() error = %v, want %v", err, expectedErr)
	}
	if !first.startInvoked {
		t.Fatal("expected first hook start to be invoked")
	}
	if !first.stopped {
		t.Fatal("expected already-started hook to be stopped on error")
	}
}
