package vm

import "context"

// DefaultMaxMetaDepth is the default __index/__newindex chain depth limit,
// matching Lua 5.4's MAXTAGLOOP. This is a safety bound to prevent infinite
// loops from metatable cycles, not a semantic limit.
const DefaultMaxMetaDepth = 2000

// Limits configures execution limits for the VM.
// Zero values mean no limit (except MaxMetaDepth, where 0 means use DefaultMaxMetaDepth).
type Limits struct {
	MaxCallDepth    int   // Maximum call stack depth (0 = unlimited)
	MaxStackSlots   int   // Maximum stack slots (0 = unlimited)
	MaxInstructions int64 // Maximum checkpoint visits (0 = unlimited)
	MaxMetaDepth    int   // Maximum __index/__newindex chain depth (0 = DefaultMaxMetaDepth)
}

// VMOption is a functional option for configuring a VM.
type VMOption func(*VM)

// WithContext returns a VMOption that sets the VM's context for cooperative cancellation.
func WithContext(ctx context.Context) VMOption {
	return func(v *VM) {
		v.ctx = ctx
	}
}

// WithLimits returns a VMOption that sets execution limits on the VM.
func WithLimits(limits Limits) VMOption {
	return func(v *VM) {
		v.limits = limits
	}
}

// WithMaxMetaDepth returns a VMOption that sets the maximum __index/__newindex
// chain depth. Values <= 0 reset to the default (DefaultMaxMetaDepth).
func WithMaxMetaDepth(n int) VMOption {
	return func(v *VM) {
		if n <= 0 {
			v.limits.MaxMetaDepth = 0
		} else {
			v.limits.MaxMetaDepth = n
		}
	}
}
