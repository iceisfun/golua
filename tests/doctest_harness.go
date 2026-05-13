package tests

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/iceisfun/golua/v1/vm"
)

// resultKind classifies how a doctest execution ended.
type resultKind int

const (
	resultSuccess  resultKind = iota // ran to completion
	resultLuaError                   // Lua error (from error(), assert failure, etc.)
	resultVMPanic                    // unexpected Go panic (VM bug)
	resultTimeout                    // context cancelled or instruction limit
)

// doctestResult captures the outcome of a single doctest run.
type doctestResult struct {
	kind resultKind
	err  error // non-nil for all kinds except resultSuccess
}

// doctestConfig holds mutable execution config that Lua helpers can adjust.
type doctestConfig struct {
	defaultTimeout time.Duration
	deadline       time.Time
	cancel         context.CancelFunc
	timer          *time.Timer
	mu             sync.Mutex
}

// doctestLimits returns the VM execution limits for doctests.
func doctestLimits() vm.Limits {
	return vm.Limits{
		MaxCallDepth:    200,
		MaxStackSlots:   10000,
		MaxInstructions: 10_000_000,
		MaxMetaDepth:    200,
	}
}

const defaultDoctestTimeout = 10 * time.Second

// registerDoctestHelpers registers the "doctest" global table with helper functions.
func registerDoctestHelpers(v *vm.VM, cfg *doctestConfig) {
	dt := vm.NewEmptyTable()

	dt.MustSet(vm.NewString("set_timeout"), vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() < 1 || !v.Get(1).IsNumber() {
			panic("doctest.set_timeout: expected positive number argument")
		}
		seconds := v.Get(1).AsFloat()
		if seconds <= 0 {
			panic("doctest.set_timeout: timeout must be positive")
		}

		cfg.mu.Lock()
		defer cfg.mu.Unlock()

		newDeadline := time.Now().Add(time.Duration(seconds * float64(time.Second)))
		if newDeadline.After(cfg.deadline) {
			panic("doctest.set_timeout: cannot raise timeout beyond default")
		}
		cfg.deadline = newDeadline
		cfg.timer.Reset(time.Until(newDeadline))
		return 0
	}))

	dt.MustSet(vm.NewString("fail"), vm.NewNativeFunc(func(v *vm.VM) int {
		msg := "doctest.fail called"
		if v.ArgCount() >= 1 && v.Get(1).IsString() {
			msg = v.Get(1).AsString()
		}
		panic(msg)
	}))

	dt.MustSet(vm.NewString("assert"), vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() < 1 {
			panic("doctest.assert: expected at least 1 argument")
		}
		val := v.Get(1)
		if !val.ToBool() {
			msg := "assertion failed"
			if v.ArgCount() >= 2 && v.Get(2).IsString() {
				msg = v.Get(2).AsString()
			}
			panic(fmt.Sprintf("doctest.assert: %s", msg))
		}
		return 0
	}))

	dt.MustSet(vm.NewString("expect_error"), vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() < 1 || !v.Get(1).IsCallable() {
			panic("doctest.expect_error: expected callable argument")
		}
		fn := v.Get(1)
		_, err := v.ProtectedCall(fn, nil)
		if err == nil {
			panic("doctest.expect_error: expected an error but call succeeded")
		}
		// Return the error value
		if le, ok := err.(*vm.LuaError); ok {
			v.Set(0, le.Value)
		} else {
			v.Set(0, vm.NewString(err.Error()))
		}
		return 1
	}))

	dt.MustSet(vm.NewString("expect_equal"), vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() < 2 {
			panic("doctest.expect_equal: expected 2 arguments")
		}
		a := v.Get(1)
		b := v.Get(2)
		if !a.Equal(b) {
			panic(fmt.Sprintf("doctest.expect_equal: %s ~= %s",
				vm.ValueToString(a), vm.ValueToString(b)))
		}
		return 0
	}))

	dt.MustSet(vm.NewString("expect_type"), vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() < 2 {
			panic("doctest.expect_type: expected 2 arguments")
		}
		val := v.Get(1)
		if !v.Get(2).IsString() {
			panic("doctest.expect_type: second argument must be a type name string")
		}
		expected := v.Get(2).AsString()
		got := val.Type()
		if got != expected {
			panic(fmt.Sprintf("doctest.expect_type: expected type %q, got %q", expected, got))
		}
		return 0
	}))

	v.SetGlobal("doctest", vm.NewTable(dt))
}

// classifyPanic determines the resultKind from a recovered panic value.
func classifyPanic(r interface{}) doctestResult {
	if r == nil {
		return doctestResult{kind: resultSuccess}
	}

	// Lua errors (from error(), assert, etc.)
	if le, ok := r.(*vm.LuaError); ok {
		return doctestResult{kind: resultLuaError, err: le}
	}

	// Check for timeout/limit messages
	msg := fmt.Sprintf("%v", r)
	if strings.Contains(msg, "instruction limit") ||
		strings.Contains(msg, "execution interrupted") ||
		strings.Contains(msg, "context canceled") ||
		strings.Contains(msg, "context deadline exceeded") {
		return doctestResult{kind: resultTimeout, err: fmt.Errorf("%s", msg)}
	}

	// Check if it's a regular error
	if err, ok := r.(error); ok {
		return doctestResult{kind: resultLuaError, err: err}
	}

	// Everything else is a VM panic (bug)
	return doctestResult{kind: resultVMPanic, err: fmt.Errorf("%v", r)}
}
