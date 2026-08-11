package vm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// LuaChanCaps declares which channel operations are exposed to Lua.
type LuaChanCaps struct {
	// AllowSend enables blocking send operations.
	AllowSend bool
	// AllowRecv enables blocking receive operations.
	AllowRecv bool
	// AllowClose enables closing channels from Lua.
	AllowClose bool
	// AllowSelect enables chan.select.
	AllowSelect bool
	// AllowTrySend enables non-blocking send operations.
	AllowTrySend bool
	// AllowTryRecv enables non-blocking receive operations.
	AllowTryRecv bool
}

// LuaChanProvider is a capability interface for Go channel operations.
// Channels are a GoLua extension (not part of standard Lua) that enable
// safe communication between Lua coroutines and Go goroutines.
type LuaChanProvider interface {
	NewChannel(ctx context.Context, size int) *LuaChannel
	Capabilities(ctx context.Context) LuaChanCaps
}

// ErrClosedChannel is returned by Send when the channel is closed, either
// before the send started or while it was blocked.
var ErrClosedChannel = errors.New("send on closed channel")

// LuaChannel wraps a Go channel for use in Lua.
type LuaChannel struct {
	id       int64
	ch       chan Value
	provider LuaChanProvider
	closed   atomic.Bool
	mu       sync.Mutex
	// done is closed by Close to wake senders blocked in their select. A Lua
	// script may close a channel a host goroutine is feeding (the producer
	// pattern in docs/chan.md), and "send on closed channel" would panic on
	// that host goroutine, where no pcall/ProtectedCall boundary exists.
	done chan struct{}
	// senders counts sends that are past the closed check. Close waits for it
	// to drain before closing the value channel, so a send can never observe
	// an already-closed channel.
	senders sync.WaitGroup
}

// enterSend registers an in-flight send unless the channel is already closed.
// The caller must call c.senders.Done() when the send finishes. The registration
// happens under c.mu so that it strictly precedes Close's Wait.
func (c *LuaChannel) enterSend() (chan struct{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed.Load() {
		return nil, false
	}
	if c.done == nil {
		// Lazily created for channels a third-party provider built directly.
		c.done = make(chan struct{})
	}
	c.senders.Add(1)
	return c.done, true
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

// Send sends a value on the channel, blocking until the send completes, the
// channel is closed, or the context is cancelled. Sending on a closed channel
// returns an error rather than panicking.
func (c *LuaChannel) Send(ctx context.Context, val Value) error {
	done, ok := c.enterSend()
	if !ok {
		return ErrClosedChannel
	}
	defer c.senders.Done()
	if ctx == nil {
		select {
		case c.ch <- val:
			return nil
		case <-done:
			return ErrClosedChannel
		}
	}
	select {
	case c.ch <- val:
		return nil
	case <-done:
		return ErrClosedChannel
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

// TrySend attempts a non-blocking send. Returns true if the value was sent;
// a closed channel reports false instead of panicking.
func (c *LuaChannel) TrySend(val Value) bool {
	if _, ok := c.enterSend(); !ok {
		return false
	}
	defer c.senders.Done()
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

// Close closes the channel. Returns an error if already closed.
func (c *LuaChannel) Close() error {
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return fmt.Errorf("close of closed channel")
	}
	c.closed.Store(true)
	if c.done == nil {
		c.done = make(chan struct{})
	}
	done := c.done
	c.mu.Unlock()

	// Wake blocked senders, then wait for every in-flight send to give up
	// before closing the value channel. Closing it underneath a sender would
	// panic on that sender's goroutine instead of returning an error.
	close(done)
	c.senders.Wait()
	close(c.ch)
	return nil
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
func (p *DefaultChanProvider) NewChannel(ctx context.Context, size int) *LuaChannel {
	id := p.nextID.Add(1)
	return &LuaChannel{
		id:       id,
		ch:       make(chan Value, size),
		provider: p,
		done:     make(chan struct{}),
	}
}

// Capabilities returns caps with all channel operations enabled.
func (p *DefaultChanProvider) Capabilities(ctx context.Context) LuaChanCaps {
	return LuaChanCaps{
		AllowSend:    true,
		AllowRecv:    true,
		AllowClose:   true,
		AllowSelect:  true,
		AllowTrySend: true,
		AllowTryRecv: true,
	}
}
