package vm

import (
	"fmt"
	"strings"
)

// Table represents a Lua table (§2.1). It has both an array part (for
// sequential integer keys 1..n) and a hash part (for all other keys).
// The keys slice maintains insertion order so that next() is deterministic.
//
// Nil-value invariant: Get, GetInt, and GetString always return a valid Value.
// They never return raw Go nil. Missing keys and deleted keys both return the
// canonical Nil value (Value{typ: typeNil}). Since Value is a struct (not a
// pointer), it is impossible for a raw Go nil to escape through these methods.
//
// The array part may contain interior holes (nil-valued slots) after
// deletion of non-trailing keys. shrinkArray removes only trailing nils.
// Next() skips nil-valued array entries so that pairs() iteration never
// exposes holes to Lua code.
//
// The hash part uses tombstone-on-nil: setting a hash key to nil removes
// the entry from the map but leaves the key in the ordered keys slice as
// a dead key. This allows ongoing pairs() iteration to skip over deleted
// entries without losing position. Dead keys are revived on re-insertion.
//
// String keys are stored in a separate strHash map, and integer keys in a
// separate intHash map, to avoid the overhead of boxing Go strings/int64s
// into the any interface required by the general hash map.
type Table struct {
	array     []Value          // sequential integer-keyed part (indices 1..n)
	hash      map[any]Value    // associative part for non-string, non-integer, non-sequential keys
	strHash   map[string]Value // associative part for string keys (avoids any boxing)
	intHash   map[int64]Value  // associative part for integer keys outside array range (avoids any boxing)
	keys      []Value          // insertion-ordered hash keys (may contain dead keys)
	deadKeys  int              // count of keys in t.keys not in t.hash/strHash/intHash
	iterBound int              // upper bound on keys slice for next() iteration (0 = no limit)
	iterHash  bool             // true after current next() traversal enters hash part
	metatable LuaTable         // per-table metatable for operator/event overrides
	isThread  bool             // true if this table represents a coroutine thread
	vmRef     *VM              // reference to coroutine VM (only set for thread tables)
	weakMode  weakTableMode    // controls whether table has weak keys, values, or both
	weak      *weakStore       // non-nil when weakMode != 0; replaces normal storage
}

// SetThread marks this table as a coroutine thread.
func (t *Table) SetThread(v bool) { t.isThread = v }

// IsThread returns whether this table represents a coroutine thread.
func (t *Table) IsThread() bool { return t.isThread }

// SetVMRef stores a reference to the coroutine VM on this thread table.
func (t *Table) SetVMRef(vm *VM) { t.vmRef = vm }

// VMRef returns the coroutine VM reference, or nil if not set.
func (t *Table) VMRef() *VM { return t.vmRef }

// maxTableEntries caps the array part of a table to bound runaway growth
// (e.g. an unbounded `t[#t+1] = v` loop inside pcall). Without this cap the
// underlying Go runtime.growslice eventually reaches its host-allocation
// limit and calls runtime.throw("out of memory"), which is uncatchable
// (recover() does not catch runtime.throw) and aborts the embedding
// process. The cap of 1<<22 (~4.2M entries) consumes ~167MB at 40B/Value
// before the cap fires; the last growslice expansion needs ~200MB of
// transient old+new headroom, well below any sane host memory budget so
// the Go allocator never reaches its throw path. The cap is far above
// any realistic Lua table workload (stress tests use 10K-class sizes).
// Once exceeded, Set/SetInt raise a Lua "not enough memory" error that
// pcall absorbs (matching reference Lua 5.5 behavior).
const maxTableEntries = 1 << 22

// NewEmptyTable creates a new empty table.
func NewEmptyTable() *Table {
	return &Table{}
}

// NewTableWithSize creates a table with preallocated array and keys space.
// The hash maps themselves are lazily allocated on first use since we cannot
// predict whether keys will be strings, integers, or other types.
func NewTableWithSize(narray, nhash int) *Table {
	t := &Table{}
	if narray > 0 {
		t.array = make([]Value, 0, narray)
	}
	if nhash > 0 {
		t.keys = make([]Value, 0, nhash)
	}
	return t
}

// EnsureArraySize grows the array part to at least n slots, filling new
// slots with Nil. This allows subsequent SetInt calls to place values
// directly into the array part even if some intermediate indices are nil.
func (t *Table) EnsureArraySize(n int) {
	if t.weak != nil {
		return // weak tables don't use array part
	}
	if n > len(t.array) {
		if n > cap(t.array) {
			newArray := make([]Value, n)
			copy(newArray, t.array)
			t.array = newArray
		} else {
			t.array = t.array[:n]
		}
	}
}

// ensureStrHash lazily initializes the string hash map.
func (t *Table) ensureStrHash() map[string]Value {
	if t.strHash == nil {
		t.strHash = make(map[string]Value)
	}
	return t.strHash
}

// ensureHash lazily initializes the general hash map.
func (t *Table) ensureHash() map[any]Value {
	if t.hash == nil {
		t.hash = make(map[any]Value)
	}
	return t.hash
}

// ensureIntHash lazily initializes the integer hash map.
func (t *Table) ensureIntHash() map[int64]Value {
	if t.intHash == nil {
		t.intHash = make(map[int64]Value)
	}
	return t.intHash
}

// setIntHash inserts, updates, or deletes an integer hash entry while keeping
// the ordered keys slice in sync.
func (t *Table) setIntHash(k int64, value Value) {
	if value.IsNil() {
		if t.intHash != nil {
			if _, exists := t.intHash[k]; exists {
				delete(t.intHash, k)
				t.deadKeys++
			}
		}
		return
	}
	ih := t.ensureIntHash()
	if _, exists := ih[k]; !exists {
		revived := false
		if t.deadKeys > 0 {
			for _, hk := range t.keys {
				if hk.typ == typeInt && hk.integer == k {
					t.deadKeys--
					revived = true
					break
				}
			}
		}
		if !revived {
			t.reuseOrAppendKey(Value{typ: typeInt, integer: k})
		}
	}
	ih[k] = value
}

// hashKey converts a Value to a map key.
func hashKey(v Value) any {
	switch v.typ {
	case typeNil:
		return nil
	case typeBool:
		return v.num != 0
	case typeInt:
		return v.integer
	case typeFloat:
		// If float is an integer, use int key for consistency
		f := v.num
		if i := int64(f); float64(i) == f {
			return i
		}
		return f
	case typeString:
		return v.ptr.(string)
	default:
		// Tables, functions use pointer identity
		return v.ptr
	}
}

// setHash inserts, updates, or deletes a non-string hash entry while keeping
// the ordered keys slice in sync. The keyValue is the original Value (used for
// the ordered keys slice), and hk is its normalized hash-key form (used as the
// Go map key, with float-that-is-int already normalized to int64 etc.).
func (t *Table) setHash(keyValue Value, hk any, value Value) {
	if value.IsNil() {
		if t.hash != nil {
			if _, exists := t.hash[hk]; exists {
				delete(t.hash, hk)
				t.deadKeys++
			}
		}
	} else {
		h := t.ensureHash()
		if _, exists := h[hk]; !exists {
			revived := false
			if t.deadKeys > 0 {
				for _, ek := range t.keys {
					if ek.RawEqual(keyValue) {
						t.deadKeys--
						revived = true
						break
					}
				}
			}
			if !revived {
				t.reuseOrAppendKey(keyValue)
			}
		}
		h[hk] = value
	}
}

// setStrHash inserts, updates, or deletes a string hash entry while keeping
// the ordered keys slice in sync.
func (t *Table) setStrHash(s string, value Value) {
	if value.IsNil() {
		if t.strHash != nil {
			if _, exists := t.strHash[s]; exists {
				delete(t.strHash, s)
				t.deadKeys++
			}
		}
		return
	}
	sh := t.ensureStrHash()
	if _, exists := sh[s]; !exists {
		revived := false
		if t.deadKeys > 0 {
			for _, hk := range t.keys {
				if hk.typ == typeString && hk.ptr.(string) == s {
					t.deadKeys--
					revived = true
					break
				}
			}
		}
		if !revived {
			t.reuseOrAppendKey(Value{typ: typeString, ptr: s})
		}
	}
	sh[s] = value
}

// reuseOrAppendKey inserts a new key into the ordered keys slice. If there is
// a dead (tombstone) slot it is overwritten in-place, which bounds the slice
// length and prevents next()-based iteration from looping infinitely when
// new keys are added during traversal. Only appends if no dead slot exists.
func (t *Table) reuseOrAppendKey(k Value) {
	if t.deadKeys > 0 {
		for i, hk := range t.keys {
			if _, alive := t.getKeyValue(hk); !alive {
				t.keys[i] = k
				t.deadKeys--
				return
			}
		}
	}
	t.keys = append(t.keys, k)
}

// removeKey removes a key from the ordered keys slice.
// Used by rehashToArray when moving keys from hash to array part.
func (t *Table) removeKey(k Value) {
	for i, existing := range t.keys {
		if existing.RawEqual(k) {
			// If this was a dead key, update the counter.
			if _, alive := t.getKeyValue(k); !alive {
				t.deadKeys--
			}
			t.keys = append(t.keys[:i], t.keys[i+1:]...)
			return
		}
	}
}

// getKeyValue retrieves the value for a key from the appropriate hash map.
func (t *Table) getKeyValue(k Value) (Value, bool) {
	switch k.typ {
	case typeString:
		if t.strHash == nil {
			return Nil, false
		}
		v, exists := t.strHash[k.ptr.(string)]
		return v, exists
	case typeInt:
		if t.intHash == nil {
			return Nil, false
		}
		v, exists := t.intHash[k.integer]
		return v, exists
	default:
		if t.hash == nil {
			return Nil, false
		}
		v, exists := t.hash[hashKey(k)]
		return v, exists
	}
}

// Get retrieves a value from the table (raw access, no metamethods).
func (t *Table) Get(key Value) Value {
	if t.weak != nil {
		return t.weak.get(key)
	}
	if key.IsNil() {
		return Nil
	}

	// Check if it's an integer key (array range or integer hash)
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok {
			if i >= 1 && int(i) <= len(t.array) {
				return t.array[i-1]
			}
			// Integer key outside array range — check integer hash
			if t.intHash != nil {
				if v, ok := t.intHash[i]; ok {
					return v
				}
			}
			return Nil
		}
	}

	// String keys use the dedicated string hash map
	if key.typ == typeString {
		if t.strHash != nil {
			if v, ok := t.strHash[key.ptr.(string)]; ok {
				return v
			}
		}
		return Nil
	}

	// General hash lookup (booleans, floats, tables, functions, etc.)
	if t.hash != nil {
		k := hashKey(key)
		if v, ok := t.hash[k]; ok {
			return v
		}
	}
	return Nil
}

// GetInt retrieves by integer key (1-based like Lua).
func (t *Table) GetInt(i int) Value {
	if t.weak != nil {
		return t.weak.get(NewInt(int64(i)))
	}
	if i >= 1 && i <= len(t.array) {
		return t.array[i-1]
	}
	if t.intHash != nil {
		if v, ok := t.intHash[int64(i)]; ok {
			return v
		}
	}
	return Nil
}

// GetString retrieves by string key.
func (t *Table) GetString(s string) Value {
	if t.weak != nil {
		return t.weak.get(NewString(s))
	}
	if t.strHash != nil {
		if v, ok := t.strHash[s]; ok {
			return v
		}
	}
	return Nil
}

// Set sets a value in the table (raw access, no metamethods).
// Returns an error for invalid keys (nil, NaN).
func (t *Table) Set(key, value Value) error {
	if t.weak != nil {
		return t.weak.set(key, value)
	}
	if key.IsNil() {
		return fmt.Errorf("table index is nil")
	}

	// Check for NaN
	if key.IsFloat() && key.num != key.num {
		return fmt.Errorf("table index is NaN")
	}

	// Check if it's an integer key (array part or integer hash)
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok {
			if i >= 1 {
				idx := int(i)
				// If within current array bounds or extending by 1
				if idx <= len(t.array) {
					if value.IsNil() {
						t.array[idx-1] = Nil
						// Shrink array if trailing nils
						t.shrinkArray()
					} else {
						t.array[idx-1] = value
					}
					return nil
				} else if idx == len(t.array)+1 && !value.IsNil() {
					// Extend array. Cap growth to bound runaway tables that
					// would otherwise reach Go's runtime.throw("out of memory")
					// from runtime.growslice (uncatchable by recover/pcall).
					if len(t.array) >= maxTableEntries {
						return fmt.Errorf("not enough memory")
					}
					t.array = append(t.array, value)
					// If this key previously existed in integer hash, clear it to
					// avoid dual storage (array + hash) for the same numeric index.
					t.setIntHash(i, Nil)
					// Move any hash entries that now belong in array
					t.rehashToArray()
					return nil
				}
			}
			// Integer key outside array range — use integer hash
			t.setIntHash(i, value)
			return nil
		}
	}

	// String keys use the dedicated string hash map
	if key.typ == typeString {
		t.setStrHash(key.ptr.(string), value)
		return nil
	}

	// Use general hash part (booleans, floats, tables, functions, etc.)
	t.setHash(key, hashKey(key), value)
	return nil
}

// MustSet is like Set but panics on error.
// For use in internal code and tests where the key is known to be valid.
func (t *Table) MustSet(key, value Value) {
	if err := t.Set(key, value); err != nil {
		panic(err)
	}
}

// SetInt sets by integer key (1-based).
func (t *Table) SetInt(i int, value Value) {
	if t.weak != nil {
		t.weak.set(NewInt(int64(i)), value)
		return
	}
	if i >= 1 && i <= len(t.array) {
		if value.IsNil() {
			t.array[i-1] = Nil
			t.shrinkArray()
		} else {
			t.array[i-1] = value
		}
		return
	}
	if i == len(t.array)+1 && !value.IsNil() {
		// Cap growth to bound runaway tables that would otherwise reach
		// Go's runtime.throw("out of memory") from runtime.growslice
		// (uncatchable by recover/pcall). SetInt has no error return so
		// raise via panic — the VM's ProtectedCall recover converts this
		// into a Lua error that pcall can catch.
		if len(t.array) >= maxTableEntries {
			panic("not enough memory")
		}
		t.array = append(t.array, value)
		// Clear stale integer-hash entry for this key to prevent duplicate
		// traversal via pairs/next after promotion into the array part.
		t.setIntHash(int64(i), Nil)
		t.rehashToArray()
		return
	}
	t.setIntHash(int64(i), value)
}

// SetString sets by string key.
func (t *Table) SetString(s string, value Value) {
	if t.weak != nil {
		t.weak.set(NewString(s), value)
		return
	}
	t.setStrHash(s, value)
}

// RawSetArray sets an array slot directly without triggering shrinkArray.
// The index must be within the current array bounds (1 <= i <= len(array)).
// This is used by table.pack to fill pre-sized arrays that may contain nils.
func (t *Table) RawSetArray(i int, value Value) {
	if t.weak != nil {
		t.weak.set(NewInt(int64(i)), value)
		return
	}
	t.array[i-1] = value
}

// ShrinkArray removes trailing nils from the array part.
func (t *Table) ShrinkArray() {
	t.shrinkArray()
}

// shrinkArray removes trailing nils from the array part.
func (t *Table) shrinkArray() {
	for len(t.array) > 0 && t.array[len(t.array)-1].IsNil() {
		t.array = t.array[:len(t.array)-1]
	}
}

// rehashToArray moves sequential integer keys from hash to array.
func (t *Table) rehashToArray() {
	for {
		nextIdx := int64(len(t.array) + 1)
		nextIdxKey := Value{typ: typeInt, integer: nextIdx}
		// Check integer hash first (most common case)
		if t.intHash != nil {
			if v, ok := t.intHash[nextIdx]; ok && !v.IsNil() {
				t.array = append(t.array, v)
				delete(t.intHash, nextIdx)
				t.removeKey(nextIdxKey)
				continue
			}
		}
		// Fall back to general hash (for legacy or float-that-is-integer keys)
		if t.hash != nil {
			if v, ok := t.hash[nextIdx]; ok && !v.IsNil() {
				t.array = append(t.array, v)
				delete(t.hash, nextIdx)
				t.removeKey(nextIdxKey)
				continue
			}
		}
		break
	}
}

// Len returns the length of the table (# operator).
// It returns a "border": an index n where t[n] is non-nil and t[n+1] is nil,
// or 0 when t[1] is nil. This matches Lua 5.4's luaH_getn.
func (t *Table) Len() int {
	if t.weak != nil {
		return t.weak.length()
	}
	n := len(t.array)
	if n == 0 {
		return 0
	}
	// Fast path: last element is non-nil → the border is at the end.
	if !t.array[n-1].IsNil() {
		// Lua 5.4 may extend the border into integer hash keys contiguous
		// after the array part.
		if t.GetInt(n + 1).IsNil() {
			return n
		}
		lo := n
		hi := n + 1
		maxInt := int(^uint(0) >> 1)
		for !t.GetInt(hi).IsNil() {
			lo = hi
			if hi > maxInt/2 {
				hi = maxInt
				break
			}
			hi *= 2
		}
		for hi-lo > 1 {
			mid := lo + (hi-lo)/2
			if t.GetInt(mid).IsNil() {
				hi = mid
			} else {
				lo = mid
			}
		}
		return lo
	}
	// Array slots can contain interior nils (e.g. constructors like
	// {nil, nil, {}, 2.5, nil}). Lua 5.4's behavior in these cases picks
	// a border near the end; scan backwards for the last non-nil entry.
	for i := n - 1; i >= 0; i-- {
		if !t.array[i].IsNil() {
			return i + 1
		}
	}
	return 0
}

// Delete removes a key from the table.
// Returns an error if the key is invalid (nil, NaN).
func (t *Table) Delete(key Value) error {
	return t.Set(key, Nil)
}

// Metatable returns the table's metatable.
func (t *Table) Metatable() LuaTable {
	return t.metatable
}

// SetMetatable sets the table's metatable.
// If the metatable has a __mode field, the table enters weak mode.
func (t *Table) SetMetatable(mt LuaTable) {
	if mt == nil {
		t.metatable = nil
		t.updateWeakMode()
		return
	}
	if tp, ok := mt.(*Table); ok && tp == nil {
		t.metatable = nil
		t.updateWeakMode()
		return
	}
	t.metatable = mt
	t.updateWeakMode()
}

// updateWeakMode parses __mode from the metatable and transitions the table
// between strong and weak storage as needed.
func (t *Table) updateWeakMode() {
	newMode := weakNone
	if t.metatable != nil {
		mode := t.metatable.Get(metaMode)
		if mode.IsString() {
			s := mode.AsString()
			hasK := strings.Contains(s, "k")
			hasV := strings.Contains(s, "v")
			if hasK && hasV {
				newMode = weakKeysValues
			} else if hasK {
				newMode = weakKeys
			} else if hasV {
				newMode = weakValues
			}
		}
	}

	if newMode == t.weakMode {
		return
	}

	// Transition: tear down old mode, set up new mode.
	if t.weakMode != weakNone {
		t.disableWeakMode()
	}
	if newMode != weakNone {
		t.enableWeakMode(newMode)
	}
	t.weakMode = newMode
}

// enableWeakMode migrates all entries from normal storage to a weakStore.
func (t *Table) enableWeakMode(mode weakTableMode) {
	ws := newWeakStore(mode)

	// Migrate array entries.
	for i, v := range t.array {
		if !v.IsNil() {
			ws.set(NewInt(int64(i+1)), v)
		}
	}
	t.array = nil

	// Migrate hash entries (using the ordered keys slice for consistency).
	for _, k := range t.keys {
		if v, alive := t.getKeyValue(k); alive {
			ws.set(k, v)
		}
	}
	t.strHash = nil
	t.intHash = nil
	t.hash = nil
	t.keys = nil
	t.deadKeys = 0

	t.weak = ws

	// Register for global sweep so ProcessGcFinalizers can implement
	// ephemeron semantics via iterative GC+sweep cycles.
	registerWeakTable(t)
}

// disableWeakMode migrates alive entries from weakStore back to normal storage.
func (t *Table) disableWeakMode() {
	if t.weak == nil {
		return
	}

	pairs := t.weak.migrate()
	t.weak = nil

	for _, p := range pairs {
		t.Set(p.k, p.v)
	}
}

// Next implements the pairs() iterator.
// Given a key, returns (nextKey, nextValue, nil).
// If key is nil, returns the first pair.
// If no more pairs, returns (Nil, Nil, nil).
// Returns an error if the key is not found in the table.
// The hash part is traversed in insertion order so that next() is
// deterministic as long as the table is not modified.
// Array entries with nil values (holes) are skipped, matching Lua semantics.
func (t *Table) Next(key Value) (Value, Value, error) {
	if t.weak != nil {
		return t.weak.next(key)
	}
	if key.IsNil() {
		// Start iteration: snapshot the hash key count so that keys added
		// during traversal don't extend iteration indefinitely.
		t.iterBound = len(t.keys)
		t.iterHash = false
		// Start iteration: find first non-nil array entry
		if kv, vv, ok := t.nextArrayEntry(0); ok {
			return kv, vv, nil
		}
		// First live hash entry
		if kv, vv, ok := t.firstLiveHashEntry(); ok {
			return kv, vv, nil
		}
		t.iterBound = 0
		return Nil, Nil, nil
	}

	// Find current key and return next.
	// Only match array-part keys if the key is exactly an integer (not a
	// float that happens to represent an integer).  Lua 5.4's next() does
	// NOT coerce float keys — next(t, 1.0) errors even when t[1] exists.
	if key.IsInt() {
		if i := key.AsInt(); i >= 1 && int(i) <= len(t.array) {
			// Currently in array part — find next non-nil entry
			if kv, vv, ok := t.nextArrayEntry(int(i)); ok {
				return kv, vv, nil
			}
			// If this numeric key was being traversed through the hash part
			// (e.g. it was promoted into array during iteration), continue from
			// its hash position instead of restarting hash traversal.
			if t.iterHash {
				if kv, vv, ok := t.nextHashAfter(key); ok {
					if kv.IsNil() {
						t.iterHash = false
					}
					return kv, vv, nil
				}
			}
			if kv, vv, ok := t.firstLiveHashEntry(); ok {
				t.iterHash = true
				return kv, vv, nil
			}
			// No more entries.
			t.iterBound = 0
			t.iterHash = false
			return Nil, Nil, nil
		}
		// The key may have been a valid array index before the array
		// shrank (trailing nils removed).  The Go slice retains its
		// capacity, so cap(t.array) reflects the former size.  If the
		// key falls within that range it was a legitimate array key
		// that was deleted; treat it as "past end of array" and
		// continue iteration into the hash part (or return nil).
		// If the key is itself a live hash entry (e.g. table.create
		// reserved an array but the key landed in the hash), advance
		// past it via nextHashAfter to avoid returning the same key
		// and looping forever.
		if i := key.AsInt(); i >= 1 && int(i) <= cap(t.array) {
			if kv, vv, ok := t.nextHashAfter(key); ok {
				if kv.IsNil() {
					t.iterHash = false
				} else {
					t.iterHash = true
				}
				return kv, vv, nil
			}
			if kv, vv, ok := t.firstLiveHashEntry(); ok {
				t.iterHash = true
				return kv, vv, nil
			}
			t.iterBound = 0
			t.iterHash = false
			return Nil, Nil, nil
		}
	}

	// Reject float keys that represent integer values.  Table storage
	// normalises such floats to integers (via hashKey), so a float like 0.0
	// would incorrectly match the stored integer key 0.  Lua 5.4's next()
	// treats float-typed keys as distinct from integer-typed keys.
	if key.IsFloat() {
		f := key.AsFloat()
		if i := int64(f); float64(i) == f {
			// This float equals an integer — it can never be a real key.
			return Nil, Nil, fmt.Errorf("invalid key to 'next'")
		}
	}

	// In hash part — find current key in ordered keys, return the next live one.
	if kv, vv, ok := t.nextHashAfter(key); ok {
		if kv.IsNil() {
			t.iterHash = false
		}
		return kv, vv, nil
	}
	t.iterHash = false
	return Nil, Nil, fmt.Errorf("invalid key to 'next'")
}

// MustNext is like Next but panics on error.
// For use in internal code and tests where the key is known to be valid.
func (t *Table) MustNext(key Value) (Value, Value) {
	k, v, err := t.Next(key)
	if err != nil {
		panic(err)
	}
	return k, v
}

// nextArrayEntry returns the first non-nil array entry at or after index start
// (0-based). Returns the Lua key, value, and true if found.
func (t *Table) nextArrayEntry(start int) (Value, Value, bool) {
	for j := start; j < len(t.array); j++ {
		if !t.array[j].IsNil() {
			return NewInt(int64(j + 1)), t.array[j], true
		}
	}
	return Nil, Nil, false
}

// firstLiveHashEntry returns the first live (non-dead) hash entry, skipping
// tombstones left by deletions during iteration. Respects iterBound so that
// keys added during iteration are not visited.
func (t *Table) firstLiveHashEntry() (Value, Value, bool) {
	bound := len(t.keys)
	if t.iterBound > 0 && t.iterBound < bound {
		bound = t.iterBound
	}
	for i := 0; i < bound; i++ {
		k := t.keys[i]
		if v, alive := t.getKeyValue(k); alive {
			return k, v, true
		}
	}
	return Nil, Nil, false
}

// nextHashAfter returns the next live hash entry after key k.
// If k exists and has no following live entries, returns (Nil, Nil, true).
// If k is not found in the current hash traversal window, returns ok=false.
func (t *Table) nextHashAfter(k Value) (Value, Value, bool) {
	bound := len(t.keys)
	if t.iterBound > 0 && t.iterBound < bound {
		bound = t.iterBound
	}
	for i := 0; i < bound; i++ {
		if !t.keys[i].RawEqual(k) {
			continue
		}
		for j := i + 1; j < bound; j++ {
			nextK := t.keys[j]
			if v, alive := t.getKeyValue(nextK); alive {
				return nextK, v, true
			}
		}
		t.iterBound = 0
		return Nil, Nil, true
	}
	return Nil, Nil, false
}

// ForEach calls fn for each key-value pair in the table.
func (t *Table) ForEach(fn func(key, value Value) bool) {
	if t.weak != nil {
		t.weak.forEach(fn)
		return
	}
	for i, v := range t.array {
		if !v.IsNil() {
			if !fn(NewInt(int64(i+1)), v) {
				return
			}
		}
	}
	for _, k := range t.keys {
		if v, alive := t.getKeyValue(k); alive {
			if !fn(k, v) {
				return
			}
		}
	}
}
