package vm

import (
	"runtime"
	"sync"
	"time"
)

// gcEntry is a pending __gc finalization queued by Go's garbage collector.
type gcEntry struct {
	table  LuaTable
	gcFunc Value
}

var (
	gcPendingMu sync.Mutex
	gcPending   []gcEntry
)

// gcTableFinalizer is the Go finalizer callback set on tables with __gc.
// Runs in Go's finalizer goroutine — cannot call Lua code directly.
// Looks up __gc from the CURRENT metatable (Lua 5.4 semantics).
func gcTableFinalizer(t *Table) {
	mt := t.Metatable()
	if mt == nil {
		return
	}
	gcFunc := mt.Get(metaGc)
	if gcFunc.IsNil() {
		return
	}
	gcPendingMu.Lock()
	gcPending = append(gcPending, gcEntry{table: t, gcFunc: gcFunc})
	gcPendingMu.Unlock()
}

// RegisterGcFinalizer checks if t has a __gc metamethod and registers a Go
// finalizer if so. Safe to call multiple times (idempotent).
//
// Lua 5.4 Reference: §2.5.3 (garbage-collection metamethods).
func RegisterGcFinalizer(t *Table) {
	mt := t.Metatable()
	if mt == nil {
		return
	}
	if mt.Get(metaGc).IsNil() {
		return
	}
	runtime.SetFinalizer(t, gcTableFinalizer)
}

// ProcessGcFinalizers triggers Go's GC, waits for finalizers to run,
// then calls pending __gc metamethods in this VM's context.
//
// When MinGCInterval is configured in VM limits, repeated calls within the
// interval skip runtime.GC() to prevent Lua scripts from causing host-level
// GC denial-of-service. Pending finalizers are always processed regardless
// of rate limiting.
func (vm *VM) ProcessGcFinalizers() {
	// Rate-limit runtime.GC() calls based on MinGCInterval.
	// Negative = disable Lua-triggered GC entirely.
	// Positive = enforce minimum interval between GC cycles.
	// Zero = no rate limit (default).
	minInterval := vm.limits.MinGCInterval
	switch {
	case minInterval < 0:
		// Lua-triggered GC disabled; skip runtime.GC()
	case minInterval > 0:
		now := time.Now()
		if vm.lastLuaGC.IsZero() || now.Sub(vm.lastLuaGC) >= minInterval {
			// Two GC cycles: first identifies unreachable objects and queues
			// their Go finalizers; second gives the finalizer goroutine a
			// chance to process them (standard Go pattern).
			runtime.GC()
			runtime.GC()
			vm.lastLuaGC = now
			vm.gcCallCount++
		}
	default:
		// No rate limit
		runtime.GC()
		runtime.GC()
		vm.gcCallCount++
	}

	gcPendingMu.Lock()
	entries := gcPending
	gcPending = nil
	gcPendingMu.Unlock()

	for _, entry := range entries {
		// Lua 5.4: __gc finalizers run inside a C-call boundary that
		// prevents yielding. Mark the context as non-yieldable.
		exit := vm.EnterNonYieldable()
		// Lua 5.4: errors in __gc are not propagated
		vm.ProtectedCall(entry.gcFunc, []Value{NewTable(entry.table)})
		exit()
	}
}
