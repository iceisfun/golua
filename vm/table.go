package vm

import "fmt"

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
type Table struct {
	array     []Value       // sequential integer-keyed part (indices 1..n)
	hash      map[any]Value // associative part for non-sequential keys
	keys      []any         // insertion-ordered hash keys (may contain dead keys)
	deadKeys  int           // count of keys in t.keys not in t.hash
	metatable LuaTable      // per-table metatable for operator/event overrides
	isThread  bool          // true if this table represents a coroutine thread
}

// SetThread marks this table as a coroutine thread.
func (t *Table) SetThread(v bool) { t.isThread = v }

// IsThread returns whether this table represents a coroutine thread.
func (t *Table) IsThread() bool { return t.isThread }

// NewEmptyTable creates a new empty table.
func NewEmptyTable() *Table {
	return &Table{
		hash: make(map[any]Value),
	}
}

// NewTableWithSize creates a table with preallocated space.
func NewTableWithSize(narray, nhash int) *Table {
	t := &Table{
		hash: make(map[any]Value, nhash),
	}
	if narray > 0 {
		t.array = make([]Value, 0, narray)
	}
	return t
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

// setHash inserts, updates, or deletes a hash entry while keeping the
// ordered keys slice in sync. Deleted keys are left in t.keys as dead
// entries (tombstones) so that ongoing pairs() iteration can skip over
// them without losing position.
func (t *Table) setHash(k any, value Value) {
	if value.IsNil() {
		if _, exists := t.hash[k]; exists {
			delete(t.hash, k)
			// Leave key in t.keys as a dead entry for iteration safety.
			t.deadKeys++
		}
	} else {
		if _, exists := t.hash[k]; !exists {
			// Check if key already exists as a dead entry in t.keys.
			revived := false
			if t.deadKeys > 0 {
				for _, hk := range t.keys {
					if hk == k {
						t.deadKeys--
						revived = true
						break
					}
				}
			}
			if !revived {
				t.keys = append(t.keys, k)
			}
		}
		t.hash[k] = value
	}
}

// removeKey removes a key from the ordered keys slice.
// Used by rehashToArray when moving keys from hash to array part.
func (t *Table) removeKey(k any) {
	for i, existing := range t.keys {
		if existing == k {
			// If this was a dead key, update the counter.
			if _, alive := t.hash[k]; !alive {
				t.deadKeys--
			}
			t.keys = append(t.keys[:i], t.keys[i+1:]...)
			return
		}
	}
}

// Get retrieves a value from the table (raw access, no metamethods).
func (t *Table) Get(key Value) Value {
	if key.IsNil() {
		return Nil
	}

	// Check if it's an integer key in array range
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok && i >= 1 && int(i) <= len(t.array) {
			return t.array[i-1]
		}
	}

	// Hash lookup
	k := hashKey(key)
	if v, ok := t.hash[k]; ok {
		return v
	}
	return Nil
}

// GetInt retrieves by integer key (1-based like Lua).
func (t *Table) GetInt(i int) Value {
	if i >= 1 && i <= len(t.array) {
		return t.array[i-1]
	}
	if v, ok := t.hash[int64(i)]; ok {
		return v
	}
	return Nil
}

// GetString retrieves by string key.
func (t *Table) GetString(s string) Value {
	if v, ok := t.hash[s]; ok {
		return v
	}
	return Nil
}

// Set sets a value in the table (raw access, no metamethods).
// Returns an error for invalid keys (nil, NaN).
func (t *Table) Set(key, value Value) error {
	if key.IsNil() {
		return fmt.Errorf("table index is nil")
	}

	// Check for NaN
	if key.IsFloat() && key.num != key.num {
		return fmt.Errorf("table index is NaN")
	}

	// Check if it's an integer key that could go in array part
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok && i >= 1 {
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
				// Extend array
				t.array = append(t.array, value)
				// Move any hash entries that now belong in array
				t.rehashToArray()
				return nil
			}
		}
	}

	// Use hash part
	t.setHash(hashKey(key), value)
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
		t.array = append(t.array, value)
		t.rehashToArray()
		return
	}
	t.setHash(int64(i), value)
}

// SetString sets by string key.
func (t *Table) SetString(s string, value Value) {
	t.setHash(s, value)
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
		if v, ok := t.hash[nextIdx]; ok && !v.IsNil() {
			t.array = append(t.array, v)
			delete(t.hash, nextIdx)
			t.removeKey(nextIdx)
		} else {
			break
		}
	}
}

// Len returns the length of the table (# operator).
// It returns a "border": an index n where t[n] is non-nil and t[n+1] is nil,
// or 0 when t[1] is nil. This matches Lua 5.4's luaH_getn.
func (t *Table) Len() int {
	n := len(t.array)
	if n == 0 {
		return 0
	}
	// Fast path: last element is non-nil → the border is at the end.
	if !t.array[n-1].IsNil() {
		return n
	}
	// Binary search for a border in [0, n).
	lo, hi := 0, n
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if t.array[mid-1].IsNil() {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
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
func (t *Table) SetMetatable(mt LuaTable) {
	if mt == nil {
		t.metatable = nil
		return
	}
	if tp, ok := mt.(*Table); ok && tp == nil {
		t.metatable = nil
		return
	}
	t.metatable = mt
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
	if key.IsNil() {
		// Start iteration: find first non-nil array entry
		if kv, vv, ok := t.nextArrayEntry(0); ok {
			return kv, vv, nil
		}
		// First live hash entry
		if kv, vv, ok := t.firstLiveHashEntry(); ok {
			return kv, vv, nil
		}
		return Nil, Nil, nil
	}

	// Find current key and return next
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok && i >= 1 && int(i) <= len(t.array) {
			// Currently in array part — find next non-nil entry
			if kv, vv, ok := t.nextArrayEntry(int(i)); ok {
				return kv, vv, nil
			}
			// No more non-nil array entries, start hash
			if kv, vv, ok := t.firstLiveHashEntry(); ok {
				return kv, vv, nil
			}
			return Nil, Nil, nil
		}
	}

	// In hash part — find current key in ordered keys, return the next live one.
	// The key may be dead (deleted during iteration) but still present in t.keys.
	k := hashKey(key)
	for i, hk := range t.keys {
		if hk == k {
			// Found key (live or dead) — advance to next live key
			for j := i + 1; j < len(t.keys); j++ {
				nextK := t.keys[j]
				if v, alive := t.hash[nextK]; alive {
					return keyToValue(nextK), v, nil
				}
			}
			return Nil, Nil, nil
		}
	}
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
// tombstones left by deletions during iteration.
func (t *Table) firstLiveHashEntry() (Value, Value, bool) {
	for _, k := range t.keys {
		if v, alive := t.hash[k]; alive {
			return keyToValue(k), v, true
		}
	}
	return Nil, Nil, false
}

// keyToValue converts a hash key back to a Value.
func keyToValue(k any) Value {
	switch v := k.(type) {
	case nil:
		return Nil
	case bool:
		return NewBool(v)
	case int64:
		return NewInt(v)
	case float64:
		return NewFloat(v)
	case string:
		return NewString(v)
	case LuaTable:
		return NewTable(v)
	case *Closure:
		return NewFunction(v)
	case NativeFunc:
		return NewNativeFunc(v)
	default:
		return Nil
	}
}

// ForEach calls fn for each key-value pair in the table.
func (t *Table) ForEach(fn func(key, value Value) bool) {
	for i, v := range t.array {
		if !v.IsNil() {
			if !fn(NewInt(int64(i+1)), v) {
				return
			}
		}
	}
	for _, k := range t.keys {
		if v, alive := t.hash[k]; alive {
			if !fn(keyToValue(k), v) {
				return
			}
		}
	}
}
