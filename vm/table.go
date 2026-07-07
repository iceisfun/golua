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
	sstr      smallStrStore    // inline string-key store; nil = unused or migrated to strHash
	intHash   map[int64]Value  // associative part for integer keys outside array range (avoids any boxing)
	keys      []Value          // insertion-ordered hash keys (may contain dead keys)
	deadKeys  int              // count of keys in t.keys not in t.hash/strHash/intHash
	iterBound int              // upper bound on keys slice for next() iteration (0 = no limit)
	iterHash  bool             // true after current next() traversal enters hash part
	metatable LuaTable         // per-table metatable for operator/event overrides
	isThread  bool             // true if this table represents a coroutine thread
	extra     *tableExtra      // lazily allocated; holds rarely-used thread/weak state
}

// tableExtra holds Table fields that only a tiny minority of tables ever use:
// the coroutine-VM back-reference (thread tables) and weak-table storage (tables
// with __mode set). Keeping them off the main Table struct shrinks the
// common-case footprint and, more importantly, removes two pointer words from
// the GC scan of every ordinary table. It is allocated lazily — `extra` stays
// nil for the overwhelming majority of tables. The hot table fields (isThread,
// metatable, the storage maps, the iteration cursor) remain inline.
type tableExtra struct {
	vmRef    *VM           // reference to coroutine VM (only set for thread tables)
	weakMode weakTableMode // controls whether table has weak keys, values, or both
	weak     *weakStore    // non-nil when weakMode != 0; replaces normal storage
}

// ensureExtra returns the table's extra block, allocating it on first use.
func (t *Table) ensureExtra() *tableExtra {
	if t.extra == nil {
		t.extra = &tableExtra{}
	}
	return t.extra
}

// weakBackend returns the table's weak storage, or nil if the table is not a
// weak table. Cheap on the common path (one pointer load, like the old
// `t.weak != nil` check).
func (t *Table) weakBackend() *weakStore {
	if t.extra != nil {
		return t.extra.weak
	}
	return nil
}

// weakModeOf returns the table's weak mode (weakNone for ordinary tables).
func (t *Table) weakModeOf() weakTableMode {
	if t.extra != nil {
		return t.extra.weakMode
	}
	return weakNone
}

// skInline is the capacity of a table's inline string-key store. A table with
// up to skInline string keys holds them in a smallStrStore (linear-scanned, no
// hashing); the (skInline+1)-th distinct string key migrates the whole set to
// the strHash map. 8 covers struct/record-shaped tables — e.g. an n-body body
// has 7 fields — without ever allocating a Go map for them.
const skInline = 8

// strEntry is one slot of a table's inline string-key store.
type strEntry struct {
	name string
	val  Value
}

// smallStrStore is a table's inline store for its first few string keys. It is
// a value-lookup accelerator only: the table's ordered keys slice (t.keys)
// still records every string key for deterministic next()/pairs iteration,
// exactly as it does for map-backed keys. Entries here are unordered (iteration
// order comes from t.keys), so deletion is an O(1) swap-remove.
//
// The store is a slice so that the very common 1-2-string-key table (e.g.
// {left=,right=} tree nodes) pays for two slots, not skInline of them. It
// starts at capacity 2 (or the NEWTABLE nhash hint, visible as cap(t.keys))
// and grows straight to skInline — one realloc, never an append cascade.
// A nil store means "no inline store" (no string keys yet, or already
// migrated to strHash); a non-nil store — even one emptied by deletions —
// means the table is in inline mode. setStrHash relies on that distinction,
// so shrinking must never replace a non-nil store with nil.
type smallStrStore []strEntry

// find returns the index of name in the store, or -1 if absent.
func (s smallStrStore) find(name string) int {
	for i := range s {
		if s[i].name == name {
			return i
		}
	}
	return -1
}

// SetThread marks this table as a coroutine thread.
func (t *Table) SetThread(v bool) { t.isThread = v }

// IsThread returns whether this table represents a coroutine thread.
func (t *Table) IsThread() bool { return t.isThread }

// SetVMRef stores a reference to the coroutine VM on this thread table.
func (t *Table) SetVMRef(vm *VM) { t.ensureExtra().vmRef = vm }

// VMRef returns the coroutine VM reference, or nil if not set.
func (t *Table) VMRef() *VM {
	if t.extra != nil {
		return t.extra.vmRef
	}
	return nil
}

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

// maxTablePrealloc caps how many slots NewTableWithSize will eagerly reserve
// from a caller-supplied size hint (e.g. table.create(narr, nrec) or a large
// table-constructor count). The hint is advisory — its only purpose is to skip
// the geometric realloc cascade while the table fills — and that benefit is
// fully amortized within a few tens of thousands of entries. Reference Lua can
// "honor" an enormous hint almost for free because it relies on lazy page
// commit; Go's make() zero-fills the whole backing store immediately, so an
// unbounded hint becomes an unbounded resident allocation that the runtime
// aborts on fatally (uncatchable by pcall). Clamping the *prealloc* well below
// maxTableEntries keeps a single huge hint to a few MB and keeps a flood of
// adversarial pcall(table.create, huge) calls comfortably ahead of GC, while a
// table that genuinely fills past this point simply grows on demand (paying the
// realloc as it goes, exactly as an un-presized table would) up to the
// maxTableEntries ceiling. 1<<16 ≈ 65536 slots ≈ 2.6MB at 40B/Value.
const maxTablePrealloc = 1 << 16

// NewEmptyTable creates a new empty table.
func NewEmptyTable() *Table {
	return &Table{}
}

// NewTableWithSize creates a table with preallocated array and keys space.
// The hash maps themselves are lazily allocated on first use since we cannot
// predict whether keys will be strings, integers, or other types.
func NewTableWithSize(narray, nhash int) *Table {
	t := &Table{}
	// narray/nhash are advisory preallocation hints. Clamp them to
	// maxTablePrealloc before any make(): a caller-supplied hint (e.g.
	// table.create(1<<29), or a giant table-constructor count) must never turn
	// into a multi-gigabyte Go backing-array allocation. Go's make zero-fills
	// the whole backing store even at len 0, forcing every page resident, so an
	// unclamped huge hint triggers runtime.throw("out of memory") — a fatal,
	// pcall-uncatchable abort. The table still grows on demand (and raises a
	// catchable "not enough memory" once it reaches the maxTableEntries ceiling).
	if narray > maxTablePrealloc {
		narray = maxTablePrealloc
	}
	if nhash > maxTablePrealloc {
		nhash = maxTablePrealloc
	}
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
	if t.weakBackend() != nil {
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
// Uses cap(t.keys) as a size hint when available so a constructor like
// {a=1,b=2,c=3} (NEWTABLE preallocates keys cap=3) creates a right-sized map.
func (t *Table) ensureStrHash() map[string]Value {
	if t.strHash == nil {
		if c := cap(t.keys); c > 0 {
			t.strHash = make(map[string]Value, c)
		} else {
			t.strHash = make(map[string]Value)
		}
	}
	return t.strHash
}

// ensureHash lazily initializes the general hash map.
func (t *Table) ensureHash() map[any]Value {
	if t.hash == nil {
		if c := cap(t.keys); c > 0 {
			t.hash = make(map[any]Value, c)
		} else {
			t.hash = make(map[any]Value)
		}
	}
	return t.hash
}

// ensureIntHash lazily initializes the integer hash map.
func (t *Table) ensureIntHash() map[int64]Value {
	if t.intHash == nil {
		if c := cap(t.keys); c > 0 {
			t.intHash = make(map[int64]Value, c)
		} else {
			t.intHash = make(map[int64]Value)
		}
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
				if hk.typ == typeInt && hk.ival() == k {
					t.deadKeys--
					revived = true
					break
				}
			}
		}
		if !revived {
			t.reuseOrAppendKey(Value{typ: typeInt, n: uint64(k)})
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
		return v.n != 0
	case typeInt:
		return v.ival()
	case typeFloat:
		// If float is an integer, use int key for consistency
		f := v.fval()
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

// setStrHash inserts, updates, or deletes a string key, keeping the ordered
// keys slice in sync. String keys live in the inline smallStrStore until the
// (skInline+1)-th distinct key arrives, at which point the whole set migrates
// to the strHash map. t.keys — the ordered key list next()/pairs depend on —
// is maintained identically in both modes.
//
// On the map path, both insert and delete detect key novelty with a len()
// compare around the single map operation rather than a separate hashed probe.
//
// boxedKey, when it is a string Value, is the caller's already-boxed form of s
// (e.g. an interpreter constant or a Set key); it is threaded through to
// addStrKey so appending a new key to t.keys need not re-box s into `any`.
// Callers that only hold the Go string pass Nil.
func (t *Table) setStrHash(s string, boxedKey, value Value) {
	if value.IsNil() {
		// Delete. The key stays in t.keys as a dead tombstone so an
		// in-progress next() keeps its position; deadKeys is bumped.
		if t.sstr != nil {
			if i := t.sstr.find(s); i >= 0 {
				last := len(t.sstr) - 1
				t.sstr[i] = t.sstr[last]
				t.sstr[last] = strEntry{} // release the string and value for GC
				t.sstr = t.sstr[:last]    // stays non-nil even when emptied
				t.deadKeys++
			}
			return
		}
		if t.strHash != nil {
			oldLen := len(t.strHash)
			delete(t.strHash, s)
			if len(t.strHash) != oldLen {
				t.deadKeys++
			}
		}
		return
	}

	// Insert or update.
	if t.sstr != nil {
		if i := t.sstr.find(s); i >= 0 {
			t.sstr[i].val = value // update in place
			return
		}
		if n := len(t.sstr); n < skInline {
			if n == cap(t.sstr) {
				// Grow straight to the full inline capacity: one realloc
				// total (2 -> 8), never a 2 -> 4 -> 8 append cascade.
				grown := make(smallStrStore, n, skInline)
				copy(grown, t.sstr)
				t.sstr = grown
			}
			t.sstr = append(t.sstr, strEntry{name: s, val: value})
			t.addStrKey(s, boxedKey)
			return
		}
		// Inline store full and s is new: migrate to the map, then fall
		// through to the map-insert path below.
		t.migrateStrStoreToMap()
	}
	if t.strHash != nil {
		oldLen := len(t.strHash)
		t.strHash[s] = value
		if len(t.strHash) != oldLen {
			t.addStrKey(s, boxedKey)
		}
		return
	}
	// First string key on this table: create the inline store. Most
	// string-keyed tables only ever hold a key or two, so start at
	// capacity 2 unless a constructor hint (NEWTABLE nhash, visible as
	// cap(t.keys)) promises more.
	c := 2
	if kc := cap(t.keys); kc > c {
		if kc > skInline {
			kc = skInline
		}
		c = kc
	}
	t.sstr = make(smallStrStore, 1, c)
	t.sstr[0] = strEntry{name: s, val: value}
	t.addStrKey(s, boxedKey)
}

// addStrKey records a newly-inserted string key in the ordered keys slice,
// reviving its dead-key tombstone if one already exists there. boxed, when it
// is a string Value, is the caller's already-boxed form of s and is stored
// as-is; otherwise s is boxed here, only on this new-key path.
func (t *Table) addStrKey(s string, boxed Value) {
	if t.deadKeys > 0 {
		for _, hk := range t.keys {
			if hk.typ == typeString && hk.ptr.(string) == s {
				t.deadKeys--
				return
			}
		}
	}
	if boxed.typ != typeString {
		boxed = Value{typ: typeString, ptr: s}
	}
	t.reuseOrAppendKey(boxed)
}

// migrateStrStoreToMap moves every entry from the inline string store into the
// strHash map and frees the store. Called when the inline store overflows.
func (t *Table) migrateStrStoreToMap() {
	sh := t.ensureStrHash()
	for i := range t.sstr {
		sh[t.sstr[i].name] = t.sstr[i].val
	}
	t.sstr = nil
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
		s := k.ptr.(string)
		if t.sstr != nil {
			if i := t.sstr.find(s); i >= 0 {
				return t.sstr[i].val, true
			}
			return Nil, false
		}
		if t.strHash == nil {
			return Nil, false
		}
		v, exists := t.strHash[s]
		return v, exists
	case typeInt:
		if t.intHash == nil {
			return Nil, false
		}
		v, exists := t.intHash[k.ival()]
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
	if ws := t.weakBackend(); ws != nil {
		return ws.get(key)
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

	// String keys: inline store first, then the string hash map.
	if key.typ == typeString {
		s := key.ptr.(string)
		if t.sstr != nil {
			if i := t.sstr.find(s); i >= 0 {
				return t.sstr[i].val
			}
			return Nil
		}
		if t.strHash != nil {
			if v, ok := t.strHash[s]; ok {
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
	if ws := t.weakBackend(); ws != nil {
		return ws.get(NewInt(int64(i)))
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
	if ws := t.weakBackend(); ws != nil {
		return ws.get(NewString(s))
	}
	if t.sstr != nil {
		if i := t.sstr.find(s); i >= 0 {
			return t.sstr[i].val
		}
		return Nil
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
	if ws := t.weakBackend(); ws != nil {
		return ws.set(key, value)
	}
	if key.IsNil() {
		return fmt.Errorf("table index is nil")
	}

	// Check for NaN. Decode the float before the self-compare: a raw bit
	// comparison (key.n != key.n) is always false and would admit NaN keys.
	if key.IsFloat() && key.fval() != key.fval() {
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
					t.growArrayForAppend()
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
		t.setStrHash(key.ptr.(string), key, value)
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
	if ws := t.weakBackend(); ws != nil {
		ws.set(NewInt(int64(i)), value)
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
		t.growArrayForAppend()
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
	if ws := t.weakBackend(); ws != nil {
		ws.set(NewString(s), value)
		return
	}
	t.setStrHash(s, Nil, value)
}

// setStringValue is SetString for callers that already hold the key in its
// boxed Value form (e.g. the interpreter's cached constant Values); it avoids
// re-boxing the key string when a new key is appended to t.keys. key must be
// a string Value.
func (t *Table) setStringValue(key, value Value) {
	if ws := t.weakBackend(); ws != nil {
		ws.set(key, value)
		return
	}
	t.setStrHash(key.ptr.(string), key, value)
}

// RawSetArray sets an array slot directly without triggering shrinkArray.
// The index must be within the current array bounds (1 <= i <= len(array)).
// This is used by table.pack to fill pre-sized arrays that may contain nils.
func (t *Table) RawSetArray(i int, value Value) {
	if ws := t.weakBackend(); ws != nil {
		ws.set(NewInt(int64(i)), value)
		return
	}
	t.array[i-1] = value
}

// growArrayForAppend makes room for one more element in a full array part.
// Small arrays grow 4x (0→4→16→64) instead of relying on append's 2x
// doubling, halving the realloc cascade for the common
// `t = {}; for i = 1, n do t[i] = v end` fill pattern; from cap 64 up,
// growth is left to append's normal policy. The explicit make is bounded
// (cap ≤ 256 Values ≈ 10 KB), far below maxTableEntries, so the runaway
// OOM cap in the callers is unaffected. Note cap(t.array) is faintly
// observable via next(): integer keys in (len, cap] are treated as
// deleted former array slots rather than "invalid key to 'next'"; this
// lenience window already existed with append slack and merely widens.
func (t *Table) growArrayForAppend() {
	if c := cap(t.array); len(t.array) == c && c < 64 {
		nc := c * 4
		if nc == 0 {
			nc = 4
		}
		na := make([]Value, len(t.array), nc)
		copy(na, t.array)
		t.array = na
	}
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
		// Respect the array-size cap. Promotion from the hash part must not
		// grow the array beyond maxTableEntries: leftover sequential keys stay
		// in the hash part (still reachable via the integer-hash get path),
		// preserving the len(t.array) <= maxTableEntries invariant that the
		// Set/SetInt append paths rely on to stay below Go's fatal-OOM ceiling.
		if len(t.array) >= maxTableEntries {
			return
		}
		nextIdx := int64(len(t.array) + 1)
		nextIdxKey := Value{typ: typeInt, n: uint64(nextIdx)}
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
// or 0 when t[1] is nil. This follows Lua 5.5's luaH_getn algorithm: probe
// near asize/2 (the synthetic hint Lua sets at array allocation time), walk
// up to 4 entries forward or backward to find a border, otherwise binary
// search. If the array part is fully populated, probe the hash part with
// a doubling search.
func (t *Table) Len() int {
	if ws := t.weakBackend(); ws != nil {
		return ws.length()
	}
	asize := len(t.array)
	if asize > 0 {
		const maxVicinity = 4
		// Lua 5.5 sets the initial lenhint to asize/2 when the array part
		// is allocated. golua does not persist a per-table hint, so we
		// recompute it here. limit must be at least 1 (a valid 1-based
		// index in the array part).
		limit := asize / 2
		if limit < 1 {
			limit = 1
		}
		if t.array[limit-1].IsNil() {
			// t[limit] empty: there must be a border before 'limit'.
			// Walk backward up to maxVicinity entries.
			for i := 0; i < maxVicinity && limit > 1; i++ {
				limit--
				if !t.array[limit-1].IsNil() {
					return limit // 'limit' is a border
				}
			}
			// Still empty; binary search [0, limit) for the border.
			lo, hi := 0, limit
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
		// t[limit] is present; walk forward looking for a border.
		for i := 0; i < maxVicinity && limit < asize; i++ {
			limit++
			if t.array[limit-1].IsNil() {
				return limit - 1 // 'limit - 1' is a border
			}
		}
		if t.array[asize-1].IsNil() {
			// Last element of array is empty; binary search [limit, asize).
			lo, hi := limit, asize
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
		// Array part is fully populated up to asize; check hash part below.
	}

	// No array part or t[asize] is non-empty; check hash part.
	if t.GetInt(asize + 1).IsNil() {
		return asize
	}
	// t[asize+1] non-empty: doubling probe + binary search in hash.
	return hashSearchBorder(t, asize)
}

// hashSearchBorder finds a border in the hash part starting from a known
// present index 'i'. Mirrors Lua 5.5's hash_search but skips the random
// seed (golua does not have luaH_getn's hint state).
func hashSearchBorder(t *Table, asize int) int {
	const maxInt = int(^uint(0) >> 1)
	i := asize + 1 // caller ensures t[i] is present
	j := i + 1
	for !t.GetInt(j).IsNil() {
		i = j
		if j > maxInt/2 {
			j = maxInt
			if t.GetInt(j).IsNil() {
				break
			}
			return j // weird case: maxInt is a boundary
		}
		j *= 2
	}
	// i present, j absent; binary search in (i, j].
	for j-i > 1 {
		mid := (i + j) / 2
		if t.GetInt(mid).IsNil() {
			j = mid
		} else {
			i = mid
		}
	}
	return i
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

	if newMode == t.weakModeOf() {
		return
	}

	// Transition: tear down old mode, set up new mode.
	if t.weakModeOf() != weakNone {
		t.disableWeakMode()
	}
	if newMode != weakNone {
		t.enableWeakMode(newMode)
	}
	t.ensureExtra().weakMode = newMode
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
	t.sstr = nil
	t.strHash = nil
	t.intHash = nil
	t.hash = nil
	t.keys = nil
	t.deadKeys = 0

	// Publish the weak backend and register for the global sweep atomically
	// under the registry lock. sweepAllWeakTables() may run concurrently in a
	// *different* VM and read this table's extra.weak pointer; without the lock
	// that read races the pointer write here. weakStore internals
	// are already guarded by ws.mu — only the pointer swap needed protection.
	publishWeakBackend(t, ws)
}

// disableWeakMode migrates alive entries from weakStore back to normal storage.
func (t *Table) disableWeakMode() {
	ws := t.weakBackend()
	if ws == nil {
		return
	}

	pairs := ws.migrate()
	// Clear the weak backend pointer under the registry lock so a concurrent
	// cross-VM sweepAllWeakTables() never reads a torn extra.weak pointer. ws.migrate() above already drained entries under ws.mu, so
	// a sweep that observed the old pointer just sweeps an empty store.
	clearWeakBackend(t)

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
	if ws := t.weakBackend(); ws != nil {
		return ws.next(key)
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
	if ws := t.weakBackend(); ws != nil {
		ws.forEach(fn)
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
