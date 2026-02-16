package vm

import "time"

// LuaTimeProvider is a capability interface for time operations.
type LuaTimeProvider interface {
	// Now returns the current time in milliseconds.
	Now() int64
}

// DefaultTimeProvider uses Go's time package.
type DefaultTimeProvider struct{}

// NewDefaultTimeProvider creates a time provider.
func NewDefaultTimeProvider() *DefaultTimeProvider {
	return &DefaultTimeProvider{}
}

// Now returns the current time in milliseconds since Unix epoch.
func (p *DefaultTimeProvider) Now() int64 {
	return time.Now().UnixMilli()
}
