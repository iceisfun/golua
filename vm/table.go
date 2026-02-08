package vm

// Table represents a Lua table.
// It has both an array part (for integer keys 1..n) and a hash part.
type Table struct {
	array     []Value
	hash      map[any]Value
	metatable *Table
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
		return int64(v.num)
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
	k := hashKey(key)
	if value.IsNil() {
		delete(t.hash, k)
	} else {
		t.hash[k] = value
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
	if value.IsNil() {
		delete(t.hash, int64(i))
	} else {
		t.hash[int64(i)] = value
	}
}

// SetString sets by string key.
func (t *Table) SetString(s string, value Value) {
	if value.IsNil() {
		delete(t.hash, s)
	} else {
		t.hash[s] = value
	}
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

// Metatable returns the table's metatable.
func (t *Table) Metatable() *Table {
	return t.metatable
}

// SetMetatable sets the table's metatable.
func (t *Table) SetMetatable(mt *Table) {
	t.metatable = mt
}

// Next implements the pairs() iterator.
// Given a key, returns (nextKey, nextValue).
// If key is nil, returns the first pair.
// If no more pairs, returns (nil, nil).
func (t *Table) Next(key Value) (Value, Value) {
	if key.IsNil() {
		// Start iteration
		if len(t.array) > 0 {
			return NewInt(1), t.array[0]
		}
		// Fall through to hash iteration
		for k, v := range t.hash {
			return keyToValue(k), v
		}
		return Nil, Nil
	}

	// Find current key and return next
	if key.IsNumber() {
		if i, ok := key.ToInt(); ok && i >= 1 && int(i) <= len(t.array) {
			// Currently in array part
			if int(i) < len(t.array) {
				return NewInt(i + 1), t.array[i]
			}
			// End of array, start hash
			for k, v := range t.hash {
				return keyToValue(k), v
			}
			return Nil, Nil
		}
	}

	// In hash part - need to find next after current key
	// This is tricky with Go maps since iteration order isn't guaranteed
	// We'll do a simple approach: collect all keys, sort, find next
	// For now, just iterate and return "some" next key
	k := hashKey(key)
	found := false
	for hk, v := range t.hash {
		if found {
			return keyToValue(hk), v
		}
		if hk == k {
			found = true
		}
	}
	return Nil, Nil
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
	for k, v := range t.hash {
		if !fn(keyToValue(k), v) {
			return
		}
	}
}
