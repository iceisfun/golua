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
	// Clear dead stack slots above vm.top before triggering GC.
	// Go's GC traces the entire vm.stack slice, but only slots below vm.top
	// are live. Without this, dead registers (e.g. from finished for-loops)
	// keep objects alive through the Go slice, preventing weak.Pointer
	// collection. This matches C Lua's behavior where the GC only traces
	// stack slots up to L->top.
	for i := vm.top; i < len(vm.stack); i++ {
		vm.stack[i] = Nil
	}

	// Rate-limit runtime.GC() calls based on MinGCInterval.
	// Negative = disable Lua-triggered GC entirely.
	// Positive = enforce minimum interval between GC cycles.
	// Zero = no rate limit (default).
	minInterval := vm.limits.MinGCInterval
	doGC := false
	switch {
	case minInterval < 0:
		// Lua-triggered GC disabled; skip runtime.GC()
	case minInterval > 0:
		now := time.Now()
		if vm.lastLuaGC.IsZero() || now.Sub(vm.lastLuaGC) >= minInterval {
			doGC = true
			vm.lastLuaGC = now
		}
	default:
		doGC = true
	}

	if doGC {
		// Two GC cycles: first identifies unreachable objects and queues
		// their Go finalizers; second gives the finalizer goroutine a
		// chance to process them (standard Go pattern).
		runtime.GC()
		runtime.GC()
		// Sweep all weak tables to remove entries with dead keys/values.
		sweepAllWeakTables()
		// Resolve ephemeron semantics: in weak-key tables, entries whose
		// keys are only reachable through other ephemeron values are removed.
		vm.resolveEphemerons()
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

// processGcFinalizersOnly processes pending __gc callbacks without doing
// any weak table sweeps or ephemeron resolution. Used by the periodic GC step
// in CheckInterrupt to avoid the overhead of full weak table processing.
func (vm *VM) processGcFinalizersOnly() {
	gcPendingMu.Lock()
	entries := gcPending
	gcPending = nil
	gcPendingMu.Unlock()

	for _, entry := range entries {
		exit := vm.EnterNonYieldable()
		vm.ProtectedCall(entry.gcFunc, []Value{NewTable(entry.table)})
		exit()
	}
}

// resolveEphemerons implements ephemeron semantics by marking tables reachable
// from roots (stack, closures, registry) WITHOUT following ephemeron table values,
// then propagating marks through live ephemeron entries' values, and finally
// removing entries whose keys are not marked. This is equivalent to the ephemeron
// mark algorithm in C Lua's GC.
func (vm *VM) resolveEphemerons() {
	if !hasEphemeronTables() {
		return
	}

	// Step 1: Collect all ephemeron key pointers.
	type ephEntry struct {
		ws     *weakStore
		entIdx int
	}
	ephKeys := make(map[*Table][]ephEntry) // key table → entries using it
	// Ephemeron resolution

	weakTablesMu.Lock()
	for _, wp := range weakTablePtrs {
		t := wp.Value()
		if t == nil || t.weak == nil || t.weak.mode != weakKeys {
			continue
		}
		ws := t.weak
		for i := range ws.entries {
			e := &ws.entries[i]
			if !e.alive || e.keyRef.isZero() {
				continue
			}
			if kt := e.keyRef.tbl.Value(); kt != nil {
				ephKeys[kt] = append(ephKeys[kt], ephEntry{ws, i})
			}
		}
	}
	weakTablesMu.Unlock()

	if len(ephKeys) == 0 {
		return
	}

	// Step 2: Mark all tables reachable from roots, skipping ephemeron values.
	live := make(map[*Table]bool)
	var markValue func(v Value)
	var markTable func(t *Table)

	markTable = func(t *Table) {
		if t == nil || live[t] {
			return
		}
		live[t] = true

		// Mark metatable.
		if mt := t.Metatable(); mt != nil {
			if ct, ok := mt.(*Table); ok {
				markTable(ct)
			}
		}

		if t.weak != nil {
			ws := t.weak
			if ws.mode == weakKeys {
				// Ephemeron table: only traverse non-collectable-keyed entries.
				for i := range ws.entries {
					e := &ws.entries[i]
					if !e.alive || !e.keyRef.isZero() {
						continue // skip collectable-keyed (ephemeron) entries
					}
					markValue(ws.entryValue(e))
				}
			} else {
				// Other weak tables: traverse all live entries.
				for i := range ws.entries {
					e := &ws.entries[i]
					if !e.alive {
						continue
					}
					k := ws.entryKey(e)
					if !k.IsNil() {
						markValue(k)
					}
					v := ws.entryValue(e)
					if !v.IsNil() {
						markValue(v)
					}
				}
			}
			return
		}

		// Normal table: traverse all entries.
		// Use raw array + hash access to avoid ForEach overhead.
		for _, v := range t.array {
			markValue(v)
		}
		for _, k := range t.keys {
			if v, ok := t.getKeyValue(k); ok {
				markValue(keyToValue(k))
				markValue(v)
			}
		}
	}

	markValue = func(v Value) {
		if v.IsTable() {
			if t, ok := v.ptr.(*Table); ok && t != nil {
				markTable(t)
			}
		} else if v.IsFunction() {
			if c, ok := v.ptr.(*Closure); ok && c != nil {
				for _, upv := range c.Upvalues {
					if upv != nil {
						markValue(upv.Get())
					}
				}
			}
		}
	}

	// Mark live stack slots per frame, using local variable scope info to
	// skip dead registers (which may hold stale references to ephemeron keys).
	for fi := range vm.callStack {
		f := &vm.callStack[fi]
		markValue(f.funcValue)
		if f.closure != nil {
			// Mark closure upvalues.
			for _, upv := range f.closure.Upvalues {
				if upv != nil {
					markValue(upv.Get())
				}
			}
			// Mark only live registers using local variable scope info.
			// In Lua's Proto.Locals, each active local at a given PC
			// occupies a register. The register index is the count of
			// active locals before it in the array (0-based).
			proto := f.closure.Proto
			pc := f.pc
			if pc > 0 {
				pc-- // frame.pc is next instruction; locals check uses current
			}
			reg := 0
			for _, loc := range proto.Locals {
				if loc.StartPC > pc {
					break
				}
				if pc < loc.EndPC {
					absIdx := f.base + reg
					if absIdx < len(vm.stack) {
						markValue(vm.stack[absIdx])
					}
					reg++
				}
			}
		} else {
			// Native frame: mark all arguments.
			for j := 1; j <= f.argc; j++ {
				absIdx := f.base + j
				if absIdx < len(vm.stack) {
					markValue(vm.stack[absIdx])
				}
			}
		}
	}

	// Mark registry.
	if vm.registry != nil {
		if rt, ok := vm.registry.(*Table); ok {
			markTable(rt)
		}
	}

	// Step 3: Propagate through ephemeron values.
	// If a key is live, its entry's value may reference more keys.
	changed := true
	for changed {
		changed = false
		for kt, entries := range ephKeys {
			if !live[kt] {
				continue
			}
			for _, ent := range entries {
				e := &ent.ws.entries[ent.entIdx]
				if !e.alive {
					continue
				}
				before := len(live)
				markValue(ent.ws.entryValue(e))
				if len(live) > before {
					changed = true
				}
			}
		}
	}

	// Step 4: Remove entries whose keys are not live.
	for kt, entries := range ephKeys {
		if live[kt] {
			continue
		}
		for _, ent := range entries {
			ent.ws.killEntry(ent.entIdx)
		}
	}
}
