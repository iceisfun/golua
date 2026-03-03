package vm

import (
	"context"
	"fmt"
	"os"

	"github.com/iceisfun/golua/compiler"
)

// DefaultMaxCallDepth is the default call depth limit, matching Lua 5.4's
// LUAI_MAXCCALLS. Prevents runaway recursion from consuming the entire Go
// goroutine stack (which is fatal and unrecoverable).
const DefaultMaxCallDepth = 200

// DefaultMaxMetaDepth is the default __index/__newindex chain depth limit,
// matching Lua 5.4's MAXTAGLOOP. This is a safety bound to prevent infinite
// loops from metatable cycles, not a semantic limit.
const DefaultMaxMetaDepth = 2000

// Limits configures execution limits for the VM.
// Zero values mean no limit (except MaxMetaDepth, where 0 means use DefaultMaxMetaDepth).
type Limits struct {
	MaxCallDepth    int                    // Maximum call stack depth (0 = DefaultMaxCallDepth, negative = unlimited)
	MaxStackSlots   int                    // Maximum stack slots (0 = unlimited)
	MaxInstructions int64                  // Maximum checkpoint visits (0 = unlimited)
	MaxMetaDepth    int                    // Maximum __index/__newindex chain depth (0 = DefaultMaxMetaDepth)
	CompilerLimits  compiler.CompilerLimits // Compiler limits passed to load()/dofile() (zero = defaults)
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

// WithCaptureOutput returns a VMOption that enables output capture.
// When enabled, Print() appends to an internal buffer instead of writing to stdout.
// Use OutputLines() to retrieve captured output.
func WithCaptureOutput(capture bool) VMOption {
	return func(v *VM) {
		v.captureOutput = capture
		if capture && v.outputLines == nil {
			buf := make([]string, 0, 64)
			v.outputLines = &buf
		}
	}
}

// Print writes a line to the print provider, captured output buffer, or stdout.
// When a LuaPrintProvider is set, output is routed to it exclusively.
// Otherwise, falls back to the capture buffer (if enabled) or stdout.
func (vm *VM) Print(line string) {
	if vm.printProvider != nil {
		vm.printProvider.Print(line)
		return
	}
	if vm.captureOutput && vm.outputLines != nil {
		*vm.outputLines = append(*vm.outputLines, line)
		return
	}
	fmt.Println(line)
}

// Warn writes a warning message to the print provider or stderr.
// When a LuaPrintProvider is set, output is routed to it exclusively.
// Otherwise, falls back to stderr.
func (vm *VM) Warn(msg string) {
	if vm.printProvider != nil {
		vm.printProvider.Warn(msg)
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

// OutputLines returns all captured output lines.
func (vm *VM) OutputLines() []string {
	if vm.outputLines == nil {
		return nil
	}
	return *vm.outputLines
}

// LastOutput returns the most recent captured output line, or "" if none.
func (vm *VM) LastOutput() string {
	if vm.outputLines == nil || len(*vm.outputLines) == 0 {
		return ""
	}
	return (*vm.outputLines)[len(*vm.outputLines)-1]
}

// ClearOutput clears all captured output lines.
func (vm *VM) ClearOutput() {
	if vm.outputLines != nil {
		*vm.outputLines = (*vm.outputLines)[:0]
	}
}
