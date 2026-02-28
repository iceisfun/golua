package vm

import (
	"runtime"
	"sync"
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
func (vm *VM) ProcessGcFinalizers() {
	// Two GC cycles: first identifies unreachable objects and queues
	// their Go finalizers; second gives the finalizer goroutine a
	// chance to process them (standard Go pattern).
	runtime.GC()
	runtime.GC()

	gcPendingMu.Lock()
	entries := gcPending
	gcPending = nil
	gcPendingMu.Unlock()

	for _, entry := range entries {
		// Lua 5.4: errors in __gc are not propagated
		vm.ProtectedCall(entry.gcFunc, []Value{NewTable(entry.table)})
	}
}
