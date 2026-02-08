package vm

// Table represents a Lua table.
// It has both an array part (for integer keys 1..n) and a hash part.
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
// The hash part uses delete-on-nil: setting a hash key to nil removes
// the entry from both the map and the ordered keys slice. There are no
// tombstone values in the hash part.
type Table struct {
	array     []Value
	hash      map[any]Value
	keys      []any // insertion-ordered hash keys
	metatable LuaTable
}

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
// ordered keys slice in sync.
func (t *Table) setHash(k any, value Value) {
	if value.IsNil() {
		if _, exists := t.hash[k]; exists {
			delete(t.hash, k)
			t.removeKey(k)
		}
	} else {
		if _, exists := t.hash[k]; !exists {
			t.keys = append(t.keys, k)
		}
		t.hash[k] = value
	}
}

// removeKey removes a key from the ordered keys slice.
func (t *Table) removeKey(k any) {
	for i, existing := range t.keys {
		if existing == k {
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
func (t *Table) Set(key, value Value) {
	if key.IsNil() {
		panic("table index is nil")
	}

	// Check for NaN
	if key.IsFloat() && key.num != key.num {
		panic("table index is NaN")
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
				return
			} else if idx == len(t.array)+1 && !value.IsNil() {
				// Extend array
				t.array = append(t.array, value)
				// Move any hash entries that now belong in array
				t.rehashToArray()
				return
			}
		}
	}

	// Use hash part
	t.setHash(hashKey(key), value)
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
// For array-like tables, this is the largest n such that t[n] is non-nil
// and t[n+1] is nil.
func (t *Table) Len() int {
	return len(t.array)
}

// Delete removes a key from the table.
func (t *Table) Delete(key Value) {
	t.Set(key, Nil)
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
// Given a key, returns (nextKey, nextValue).
// If key is nil, returns the first pair.
// If no more pairs, returns (nil, nil).
// The hash part is traversed in insertion order so that next() is
// deterministic as long as the table is not modified.
// Array entries with nil values (holes) are skipped, matching Lua semantics.
func (t *Table) Next(key Value) (Value, Value) {
	if key.IsNil() {
		// Start iteration: find first non-nil array entry
		if kv, vv, ok := t.nextArrayEntry(0); ok {
			return kv, vv
		}
		// First hash entry
		if len(t.keys) > 0 {
			k := t.keys[0]
			return keyToValue(k), t.hash[k]
		}
		return Nil, Nil
	}

	// Find current key and return next
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok && i >= 1 && int(i) <= len(t.array) {
			// Currently in array part — find next non-nil entry
			if kv, vv, ok := t.nextArrayEntry(int(i)); ok {
				return kv, vv
			}
			// No more non-nil array entries, start hash
			if len(t.keys) > 0 {
				k := t.keys[0]
				return keyToValue(k), t.hash[k]
			}
			return Nil, Nil
		}
	}

	// In hash part — find current key in ordered keys, return the next one
	k := hashKey(key)
	for i, hk := range t.keys {
		if hk == k {
			if i+1 < len(t.keys) {
				nextK := t.keys[i+1]
				return keyToValue(nextK), t.hash[nextK]
			}
			return Nil, Nil
		}
	}
	return Nil, Nil
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
	default:
		// Table or function pointer - we can't easily recover this
		// This is a limitation of our simple Next implementation
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
		if !fn(keyToValue(k), t.hash[k]) {
			return
		}
	}
}
