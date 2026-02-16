package vm

import (
	"sync"
	"time"
)

// LuaTimeProvider is a capability interface for time operations.
type LuaTimeProvider interface {
	// Now returns the current time in milliseconds.
	Now() int64

	// Tick returns true once per interval (ms) for the given key,
	// false otherwise. Used for periodic logic in hot paths.
	Tick(key string, ms int64) bool
}

// DefaultTimeProvider uses Go's time package.
type DefaultTimeProvider struct {
	mu    sync.Mutex
	ticks map[string]int64
}

// NewDefaultTimeProvider creates a time provider.
func NewDefaultTimeProvider() *DefaultTimeProvider {
	return &DefaultTimeProvider{
		ticks: make(map[string]int64),
	}
}

// Now returns the current time in milliseconds since Unix epoch.
func (p *DefaultTimeProvider) Now() int64 {
	return time.Now().UnixMilli()
}

// Tick returns true if at least ms milliseconds have elapsed since the
// last true for this key. First call for a key always returns true.
func (p *DefaultTimeProvider) Tick(key string, ms int64) bool {
	now := time.Now().UnixMilli()
	p.mu.Lock()
	defer p.mu.Unlock()
	last, ok := p.ticks[key]
	if !ok || now-last >= ms {
		p.ticks[key] = now
		return true
	}
	return false
}
