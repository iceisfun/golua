package vm

import "context"

// Limits configures execution limits for the VM.
// Zero values mean no limit.
type Limits struct {
	MaxCallDepth    int   // Maximum call stack depth (0 = unlimited)
	MaxStackSlots   int   // Maximum stack slots (0 = unlimited)
	MaxInstructions int64 // Maximum checkpoint visits (0 = unlimited)
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
