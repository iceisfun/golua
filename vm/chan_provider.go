package vm

import (
	"context"
	"sync"
	"sync/atomic"
)

// LuaChanCaps declares which channel operations are allowed.
type LuaChanCaps struct {
	AllowSend    bool
	AllowRecv    bool
	AllowClose   bool
	AllowSelect  bool
	AllowTrySend bool
	AllowTryRecv bool
}

// LuaChanProvider is a capability interface for Go channel operations.
// Channels are a GoLua extension (not part of standard Lua) that enable
// safe communication between Lua coroutines and Go goroutines.
type LuaChanProvider interface {
	NewChannel(size int) *LuaChannel
	Capabilities() LuaChanCaps
}

// LuaChannel wraps a Go channel for use in Lua.
type LuaChannel struct {
	id       int64
	ch       chan Value
	provider LuaChanProvider
	closed   atomic.Bool
	mu       sync.Mutex
}

// ID returns the channel's unique identifier.
func (c *LuaChannel) ID() int64 {
	return c.id
}

// Provider returns the provider that created this channel.
func (c *LuaChannel) Provider() LuaChanProvider {
	return c.provider
}

// GoChan returns the underlying Go channel for use with reflect.Select or Go-side operations.
func (c *LuaChannel) GoChan() chan Value {
	return c.ch
}

// IsClosed returns whether the channel has been closed.
func (c *LuaChannel) IsClosed() bool {
	return c.closed.Load()
}

// Send sends a value on the channel, blocking until the send completes or the context is cancelled.
func (c *LuaChannel) Send(ctx context.Context, val Value) error {
	if ctx == nil {
		c.ch <- val
		return nil
	}
	select {
	case c.ch <- val:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Recv receives a value from the channel, blocking until a value is available or the context is cancelled.
// Returns (value, ok, error) where ok is false if the channel is closed and drained.
func (c *LuaChannel) Recv(ctx context.Context) (Value, bool, error) {
	if ctx == nil {
		val, ok := <-c.ch
		return val, ok, nil
	}
	select {
	case val, ok := <-c.ch:
		return val, ok, nil
	case <-ctx.Done():
		return Nil, false, ctx.Err()
	}
}

// TrySend attempts a non-blocking send. Returns true if the value was sent.
func (c *LuaChannel) TrySend(val Value) bool {
	select {
	case c.ch <- val:
		return true
	default:
		return false
	}
}

// TryRecv attempts a non-blocking receive.
// Returns (value, ok, received) where:
//   - received=true, ok=true: got a value
//   - received=true, ok=false: channel closed and drained
//   - received=false: would block (no data, channel still open)
func (c *LuaChannel) TryRecv() (Value, bool, bool) {
	select {
	case val, ok := <-c.ch:
		return val, ok, true
	default:
		return Nil, false, false
	}
}

// Close closes the channel. Panics if already closed.
func (c *LuaChannel) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed.Load() {
		panic("close of closed channel")
	}
	c.closed.Store(true)
	close(c.ch)
}

// DefaultChanProvider enables all channel capabilities with atomic ID generation.
type DefaultChanProvider struct {
	nextID atomic.Int64
}

// NewDefaultChanProvider creates a channel provider with all capabilities enabled.
func NewDefaultChanProvider() *DefaultChanProvider {
	return &DefaultChanProvider{}
}

// NewChannel creates a new channel with the given buffer size.
func (p *DefaultChanProvider) NewChannel(size int) *LuaChannel {
	id := p.nextID.Add(1)
	return &LuaChannel{
		id:       id,
		ch:       make(chan Value, size),
		provider: p,
	}
}

// Capabilities returns caps with all channel operations enabled.
func (p *DefaultChanProvider) Capabilities() LuaChanCaps {
	return LuaChanCaps{
		AllowSend:    true,
		AllowRecv:    true,
		AllowClose:   true,
		AllowSelect:  true,
		AllowTrySend: true,
		AllowTryRecv: true,
	}
}
