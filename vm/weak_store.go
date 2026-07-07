package vm

import (
	"fmt"
	"reflect"
	"sync"
	"weak"
)

// weakTableMode controls the weakness semantics of a table.
type weakTableMode byte

const (
	weakNone       weakTableMode = 0   // strong table (default)
	weakKeys       weakTableMode = 'k' // weak keys only
	weakValues     weakTableMode = 'v' // weak values only
	weakKeysValues weakTableMode = 'b' // both weak keys and weak values
)

// weakStore is an alternative table storage backend for tables with __mode set.
// It uses weak references for the appropriate dimension (keys, values, or both)
// so that entries with only-weakly-reachable keys or values are eventually
// cleaned up by Go's garbage collector.
//
// Unlike the normal Table storage (array + 3 typed hash maps), weakStore uses
// a single ordered entries list with a lookup index. This trades some
// normal-path performance for correct weak semantics.
type weakStore struct {
	// mu guards all fields. A weakStore is normally accessed only by its owning
	// VM (whose coroutines run serialized via channel handoff), but the global
	// sweepAllWeakTables() runs from whichever VM happens to call collectgarbage
	// and may touch a store owned by a *different*, concurrently-running VM.
	// mu makes that cross-VM access race-free.
	mu sync.Mutex

	mode weakTableMode

	entries []weakEntry
	index   map[any]int // lookup key → entry index

	dead      int // count of dead (tombstone) entries
	iterBound int // snapshot of len(entries) at iteration start
}

// weakEntry stores a single key-value pair with optional weak tracking.
type weakEntry struct {
	// Key: for value-type keys, key holds the Value directly.
	// For collectable keys in weak-key mode, key is Nil and keyRef tracks liveness.
	key    Value
	keyRef weakRef // non-zero for collectable keys in weak-key mode

	// Value: for value-type values, value holds the Value directly.
	// For collectable values in weak-value mode, value is Nil and valueRef tracks liveness.
	value    Value
	valueRef weakRef // non-zero for collectable values in weak-value mode

	// indexKey is the key used in ws.index for this entry, retained so that
	// killEntry can remove it without recomputing (which might fail for dead keys).
	indexKey any

	alive bool
}

// newWeakStore creates a new weak store with the given mode.
func newWeakStore(mode weakTableMode) *weakStore {
	return &weakStore{
		mode:  mode,
		index: make(map[any]int),
	}
}

// weakKeyFor returns the lookup key for the index map.
// For weak-key tables with collectable keys, uses uintptr to avoid
// holding a strong reference that would prevent garbage collection.
func (ws *weakStore) weakKeyFor(key Value) any {
	if ws.hasWeakKeys() && isCollectable(key) {
		return reflect.ValueOf(key.ptr).Pointer()
	}
	return hashKey(key)
}

// hasWeakKeys reports whether this store uses weak keys.
func (ws *weakStore) hasWeakKeys() bool {
	return ws.mode == weakKeys || ws.mode == weakKeysValues
}

// hasWeakValues reports whether this store uses weak values.
func (ws *weakStore) hasWeakValues() bool {
	return ws.mode == weakValues || ws.mode == weakKeysValues
}

// get retrieves the value for a key, returning Nil if not found or dead.
func (ws *weakStore) get(key Value) Value {
	if key.IsNil() {
		return Nil
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()

	wk := ws.weakKeyFor(key)
	idx, ok := ws.index[wk]
	if !ok {
		return Nil
	}

	e := &ws.entries[idx]
	if !e.alive {
		return Nil
	}

	// Check key liveness for weak-key tables.
	if !e.keyRef.isZero() && !e.keyRef.alive() {
		ws.killEntry(idx)
		return Nil
	}

	// Return value, checking liveness for weak-value tables.
	return ws.entryValue(e)
}

// set stores a key-value pair, handling weak references as appropriate.
func (ws *weakStore) set(key, value Value) error {
	if key.IsNil() {
		return fmt.Errorf("table index is nil")
	}
	// Decode the float before the NaN self-compare: a raw bit comparison
	// (key.n != key.n) is always false and would admit NaN keys.
	if key.IsFloat() && key.fval() != key.fval() {
		return fmt.Errorf("table index is NaN")
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()

	wk := ws.weakKeyFor(key)

	if value.IsNil() {
		// Delete entry.
		if idx, ok := ws.index[wk]; ok {
			ws.killEntry(idx)
		}
		return nil
	}

	// Update existing entry.
	if idx, ok := ws.index[wk]; ok {
		e := &ws.entries[idx]
		if !e.alive {
			// Revive dead entry.
			e.alive = true
			ws.dead--
		}
		ws.setEntryValue(e, value)
		return nil
	}

	// New entry.
	e := weakEntry{alive: true, indexKey: wk}
	ws.setEntryKey(&e, key)
	ws.setEntryValue(&e, value)

	// Try to reuse a dead slot to bound entries growth.
	if ws.dead > 0 {
		for i := range ws.entries {
			if !ws.entries[i].alive {
				ws.entries[i] = e
				ws.index[wk] = i
				ws.dead--
				return nil
			}
		}
	}

	ws.index[wk] = len(ws.entries)
	ws.entries = append(ws.entries, e)
	return nil
}

// setEntryKey stores the key in the entry, using a weak reference for
// collectable keys in weak-key mode.
func (ws *weakStore) setEntryKey(e *weakEntry, key Value) {
	if ws.hasWeakKeys() && isCollectable(key) {
		ref, _ := makeWeakRef(key)
		e.keyRef = ref
		e.key = Nil // don't hold strong reference
	} else {
		e.key = key
		e.keyRef = weakRef{}
	}
}

// setEntryValue stores the value in the entry, using a weak reference for
// collectable values in weak-value mode.
func (ws *weakStore) setEntryValue(e *weakEntry, value Value) {
	if ws.hasWeakValues() && isCollectable(value) {
		ref, _ := makeWeakRef(value)
		e.valueRef = ref
		e.value = Nil // don't hold strong reference
	} else {
		e.value = value
		e.valueRef = weakRef{}
	}
}

// entryKey returns the live key for an entry, or Nil if the key was collected.
func (ws *weakStore) entryKey(e *weakEntry) Value {
	if !e.keyRef.isZero() {
		if kv, ok := e.keyRef.value(); ok {
			return kv
		}
		return Nil
	}
	return e.key
}

// entryValue returns the live value for an entry, or Nil if the value was collected.
func (ws *weakStore) entryValue(e *weakEntry) Value {
	if !e.valueRef.isZero() {
		if vv, ok := e.valueRef.value(); ok {
			return vv
		}
		return Nil
	}
	return e.value
}

// killEntry marks entry at idx as dead and removes it from the index.
func (ws *weakStore) killEntry(idx int) {
	e := &ws.entries[idx]
	if !e.alive {
		return
	}
	e.alive = false
	e.key = Nil
	e.value = Nil
	e.keyRef = weakRef{}
	e.valueRef = weakRef{}
	ws.dead++
	delete(ws.index, e.indexKey)
}

// next implements the pairs() iterator for weak tables.
func (ws *weakStore) next(key Value) (Value, Value, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if key.IsNil() {
		// Start iteration.
		ws.iterBound = len(ws.entries)
		return ws.nextAliveFrom(0)
	}

	// Find current key in index.
	wk := ws.weakKeyFor(key)
	idx, ok := ws.index[wk]
	if !ok {
		return Nil, Nil, fmt.Errorf("invalid key to 'next'")
	}

	// Return next alive entry after idx.
	return ws.nextAliveFrom(idx + 1)
}

// nextAliveFrom returns the next alive entry at or after start index.
func (ws *weakStore) nextAliveFrom(start int) (Value, Value, error) {
	bound := ws.iterBound
	if bound == 0 {
		bound = len(ws.entries)
	}
	if bound > len(ws.entries) {
		bound = len(ws.entries)
	}

	for i := start; i < bound; i++ {
		e := &ws.entries[i]
		if !e.alive {
			continue
		}

		k := ws.entryKey(e)
		if k.IsNil() {
			// Key was collected.
			ws.killEntry(i)
			continue
		}

		v := ws.entryValue(e)
		if v.IsNil() {
			// Value was collected.
			ws.killEntry(i)
			continue
		}

		return k, v, nil
	}

	ws.iterBound = 0
	return Nil, Nil, nil
}

// length returns the sequence length (# operator) for the weak table.
func (ws *weakStore) length() int {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	n := 0
	for {
		nextKey := int64(n + 1)
		// Integer keys are value types, always use int64 as lookup key.
		idx, ok := ws.index[nextKey]
		if !ok {
			break
		}
		e := &ws.entries[idx]
		if !e.alive {
			break
		}
		v := ws.entryValue(e)
		if v.IsNil() {
			ws.killEntry(idx)
			break
		}
		n++
	}
	return n
}

// forEach calls fn for each live key-value pair in insertion order.
//
// The live pairs are snapshotted under the lock and fn is invoked outside it,
// because fn runs arbitrary Lua code that may re-enter this same weakStore
// (which would deadlock the non-reentrant mutex).
func (ws *weakStore) forEach(fn func(key, value Value) bool) {
	ws.mu.Lock()
	type kv struct{ k, v Value }
	pairs := make([]kv, 0, len(ws.entries)-ws.dead)
	for i := range ws.entries {
		e := &ws.entries[i]
		if !e.alive {
			continue
		}

		k := ws.entryKey(e)
		if k.IsNil() {
			ws.killEntry(i)
			continue
		}

		v := ws.entryValue(e)
		if v.IsNil() {
			ws.killEntry(i)
			continue
		}
		pairs = append(pairs, kv{k, v})
	}
	ws.mu.Unlock()

	for _, p := range pairs {
		if !fn(p.k, p.v) {
			return
		}
	}
}

// sweep removes all dead entries whose weak references have been collected and
// returns the number of live entries it killed.
func (ws *weakStore) sweep() int {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	before := len(ws.entries) - ws.dead
	for i := range ws.entries {
		e := &ws.entries[i]
		if !e.alive {
			continue
		}

		if !e.keyRef.isZero() && !e.keyRef.alive() {
			ws.killEntry(i)
			continue
		}

		if !e.valueRef.isZero() && !e.valueRef.alive() {
			ws.killEntry(i)
		}
	}
	return before - (len(ws.entries) - ws.dead)
}

// migrate copies all live entries out as key-value pairs for transfer back
// to normal table storage.
func (ws *weakStore) migrate() []struct{ k, v Value } {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	var pairs []struct{ k, v Value }
	for i := range ws.entries {
		e := &ws.entries[i]
		if !e.alive {
			continue
		}
		k := ws.entryKey(e)
		v := ws.entryValue(e)
		if k.IsNil() || v.IsNil() {
			continue
		}
		pairs = append(pairs, struct{ k, v Value }{k, v})
	}
	return pairs
}

// Global weak table registry — tracks all tables with weak stores so that
// ProcessGcFinalizers can sweep them after GC cycles.
var (
	weakTablesMu  sync.Mutex
	weakTablePtrs []weak.Pointer[Table]
)

// publishWeakBackend atomically sets t's weak backend and registers the table
// for the global sweep, holding weakTablesMu so a concurrent cross-VM
// sweepAllWeakTables() never reads a torn extra.weak pointer.
func publishWeakBackend(t *Table, ws *weakStore) {
	weakTablesMu.Lock()
	defer weakTablesMu.Unlock()
	t.ensureExtra().weak = ws
	weakTablePtrs = append(weakTablePtrs, weak.Make(t))
}

// clearWeakBackend atomically clears t's weak backend under weakTablesMu, so the
// pointer write does not race a concurrent cross-VM sweep's read.
func clearWeakBackend(t *Table) {
	weakTablesMu.Lock()
	defer weakTablesMu.Unlock()
	t.extra.weak = nil
}

// sweepAllWeakTables sweeps all registered weak tables, removing entries with
// collected keys or values. Returns the number of entries removed.
func sweepAllWeakTables() int {
	weakTablesMu.Lock()
	defer weakTablesMu.Unlock()
	removed := 0
	j := 0
	for i := range weakTablePtrs {
		t := weakTablePtrs[i].Value()
		if t == nil {
			continue // table itself was collected
		}
		weakTablePtrs[j] = weakTablePtrs[i]
		j++
		ws := t.weakBackend()
		if ws == nil {
			continue
		}
		removed += ws.sweep()
	}
	for i := j; i < len(weakTablePtrs); i++ {
		weakTablePtrs[i] = weak.Pointer[Table]{}
	}
	weakTablePtrs = weakTablePtrs[:j]
	return removed
}
