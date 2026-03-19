package vm

import (
	"context"
	"sync"
	"time"
)

// LuaTimeProvider is a capability interface for millisecond timing operations.
// This is a GoLua extension (not part of standard Lua) providing high-resolution
// timing primitives for game loops, rate limiting, and initialization guards.
type LuaTimeProvider interface {
	// Now returns the current time in milliseconds.
	Now(ctx context.Context) int64

	// Tick returns true once per interval (ms) for the given key,
	// false otherwise. Used for periodic logic in hot paths.
	Tick(ctx context.Context, key string, ms int64) bool

	// Once returns true on the first call for a given key, false
	// on all subsequent calls. Used for one-time initialization.
	Once(ctx context.Context, key string) bool
}

// DefaultTimeProvider uses Go's time package.
type DefaultTimeProvider struct {
	mu    sync.Mutex
	ticks map[string]int64
	onces map[string]bool
}

// NewDefaultTimeProvider creates a time provider.
func NewDefaultTimeProvider() *DefaultTimeProvider {
	return &DefaultTimeProvider{
		ticks: make(map[string]int64),
		onces: make(map[string]bool),
	}
}

// Now returns the current time in milliseconds since Unix epoch.
func (p *DefaultTimeProvider) Now(ctx context.Context) int64 {
	return time.Now().UnixMilli()
}

const (
	maxTickKeys   = 10000
	maxTickKeyLen = 512
)

// Tick returns true if at least ms milliseconds have elapsed since the
// last true for this key. First call for a key always returns true.
// Keys longer than 512 bytes are truncated. Once 10,000 distinct keys
// exist, new keys are silently ignored (returns false).
func (p *DefaultTimeProvider) Tick(ctx context.Context, key string, ms int64) bool {
	if len(key) > maxTickKeyLen {
		key = key[:maxTickKeyLen]
	}
	now := time.Now().UnixMilli()
	p.mu.Lock()
	defer p.mu.Unlock()
	last, ok := p.ticks[key]
	if !ok {
		if len(p.ticks) >= maxTickKeys {
			return false
		}
		p.ticks[key] = now
		return true
	}
	if now-last >= ms {
		p.ticks[key] = now
		return true
	}
	return false
}

// Once returns true on the first call for a given key, false on all
// subsequent calls. Keys longer than 512 bytes are truncated. Once
// 10,000 distinct keys exist, new keys are silently ignored (returns false).
func (p *DefaultTimeProvider) Once(ctx context.Context, key string) bool {
	if len(key) > maxTickKeyLen {
		key = key[:maxTickKeyLen]
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.onces[key] {
		return false
	}
	if len(p.onces) >= maxTickKeys {
		return false
	}
	p.onces[key] = true
	return true
}
