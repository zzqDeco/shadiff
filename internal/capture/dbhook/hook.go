package dbhook

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"shadiff/internal/model"
)

const (
	DefaultFlushTimeout  = 200 * time.Millisecond
	hookReadPollInterval = 10 * time.Millisecond
	hookFlushIdleWindow  = 10 * time.Millisecond
)

// DBHook is the interface for capturing database operations
type DBHook interface {
	// Start starts the database proxy, listening on listenAddr and forwarding to targetAddr
	Start(ctx context.Context) error
	// Flush blocks until traffic already observed by the hook has been parsed into SideEffects().
	Flush(ctx context.Context) error
	// Stop stops the proxy
	Stop() error
	// SideEffects returns the channel of captured side effects
	SideEffects() <-chan model.SideEffect
	// Type returns the database type
	Type() string
}

type activeConn struct {
	client  net.Conn
	server  net.Conn
	flushCh chan chan struct{}
	done    chan struct{}
}

type sideEffectForwarder struct {
	source   <-chan model.SideEffect
	sink     chan<- model.SideEffect
	done     chan struct{}
	barrier  chan chan struct{}
	inFlight atomic.Int64
}

func newSideEffectForwarder(ctx context.Context, source <-chan model.SideEffect, sink chan<- model.SideEffect) *sideEffectForwarder {
	f := &sideEffectForwarder{
		source:  source,
		sink:    sink,
		done:    make(chan struct{}),
		barrier: make(chan chan struct{}),
	}

	go func() {
		defer close(f.done)
		for {
			select {
			case effect, ok := <-source:
				if !ok {
					return
				}
				f.inFlight.Add(1)
				select {
				case sink <- effect:
				case <-ctx.Done():
					f.inFlight.Add(-1)
					return
				}
				f.inFlight.Add(-1)
			case ack := <-f.barrier:
				close(ack)
			case <-ctx.Done():
				return
			}
		}
	}()

	return f
}

func (f *sideEffectForwarder) WaitDrained(ctx context.Context) error {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		if len(f.source) == 0 && f.inFlight.Load() == 0 {
			if err := f.waitBarrier(ctx); err != nil {
				return err
			}
			if len(f.source) == 0 && f.inFlight.Load() == 0 {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		case <-f.done:
			if f.inFlight.Load() == 0 {
				return nil
			}
		}
	}
}

func (f *sideEffectForwarder) waitBarrier(ctx context.Context) error {
	ack := make(chan struct{})
	select {
	case f.barrier <- ack:
	case <-ctx.Done():
		return ctx.Err()
	case <-f.done:
		if f.inFlight.Load() == 0 {
			return nil
		}
	}

	select {
	case <-ack:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-f.done:
		if f.inFlight.Load() == 0 {
			return nil
		}
		return nil
	}
}

// Group coordinates a set of DB hooks and their side-effect forwarders.
type Group struct {
	hooks      []DBHook
	forwarders []*sideEffectForwarder
}

// NewGroup creates a hook group and starts forwarding each hook's side effects into sink.
func NewGroup(ctx context.Context, hooks []DBHook, sink chan<- model.SideEffect) *Group {
	group := &Group{
		hooks:      append([]DBHook(nil), hooks...),
		forwarders: make([]*sideEffectForwarder, 0, len(hooks)),
	}
	for _, hook := range hooks {
		group.forwarders = append(group.forwarders, newSideEffectForwarder(ctx, hook.SideEffects(), sink))
	}
	return group
}

// Flush waits for each hook to parse already-seen traffic and for forwarded side effects to reach sink.
func (g *Group) Flush(ctx context.Context) error {
	var firstErr error
	for _, hook := range g.hooks {
		if err := hook.Flush(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, forwarder := range g.forwarders {
		if err := forwarder.WaitDrained(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Stop stops all hooks in the group.
func (g *Group) Stop() error {
	var firstErr error
	for _, hook := range g.hooks {
		if err := hook.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Config is the common configuration for DB proxies
type Config struct {
	DBType     string // mysql / postgres / mongo / redis
	ListenAddr string // proxy listen address
	TargetAddr string // real DB address
}

// NewHook creates the corresponding DB hook based on the type
func NewHook(cfg Config) (DBHook, error) {
	switch cfg.DBType {
	case "mysql":
		return NewMySQLHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "postgres":
		return NewPostgresHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "mongo":
		return NewMongoHook(cfg.ListenAddr, cfg.TargetAddr), nil
	case "redis":
		return NewRedisHook(cfg.ListenAddr, cfg.TargetAddr), nil
	default:
		return nil, &UnsupportedDBError{DBType: cfg.DBType}
	}
}

// UnsupportedDBError represents an unsupported database type error
type UnsupportedDBError struct {
	DBType string
}

func (e *UnsupportedDBError) Error() string {
	return "unsupported database type: " + e.DBType
}
