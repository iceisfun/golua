package vm

// LuaTable is the interface for Lua table values. The concrete [Table] type
// implements this interface. Custom implementations may be provided for
// specialized use cases (virtual filesystems, proxy objects, etc.).
//
// All methods follow Lua 5.4 raw access semantics — metamethods are NOT
// invoked by Get/Set. The VM handles metamethod dispatch separately.
//
// Lua 5.4 Reference: §2.1 (tables), §6.1 (table library).
type LuaTable interface {
	// Get returns the value associated with key, or Nil if absent.
	Get(key Value) Value
	// Set assigns val to key. Setting a key to Nil removes it.
	// Returns an error for invalid keys (nil, NaN).
	Set(key Value, val Value) error
	// Delete removes the given key from the table.
	// Returns an error if the key is invalid (nil, NaN).
	Delete(key Value) error
	// Next returns the next key-value pair after key for table traversal.
	// Pass Nil as key to get the first pair. Returns (Nil, Nil, nil) at end.
	// Returns an error if key is not in the table.
	Next(key Value) (nextKey Value, val Value, err error)
	// Len returns the raw length of the table's sequence part.
	Len() int
	// Metatable returns the table's metatable, or nil if none is set.
	Metatable() LuaTable
	// SetMetatable sets (or clears) the table's metatable.
	SetMetatable(mt LuaTable)
}

// Compile-time check that *Table implements LuaTable.
var _ LuaTable = (*Table)(nil)

// Pre-allocated metamethod name values. These are created once at init time
// to avoid repeated string allocation in hot paths like metamethod lookups.
// The full set covers all Lua 5.4 metamethods (§2.4).
var (
	metaIndex     = NewString("__index")
	metaNewIndex  = NewString("__newindex")
	metaClose     = NewString("__close")
	metaCall      = NewString("__call")
	metaAdd       = NewString("__add")
	metaSub       = NewString("__sub")
	metaMul       = NewString("__mul")
	metaDiv       = NewString("__div")
	metaMod       = NewString("__mod")
	metaPow       = NewString("__pow")
	metaUnm       = NewString("__unm")
	metaIDiv      = NewString("__idiv")
	metaBand      = NewString("__band")
	metaBor       = NewString("__bor")
	metaBxor      = NewString("__bxor")
	metaBnot      = NewString("__bnot")
	metaShl       = NewString("__shl")
	metaShr       = NewString("__shr")
	metaEq        = NewString("__eq")
	metaLt        = NewString("__lt")
	metaLe        = NewString("__le")
	metaLen       = NewString("__len")
	metaConcat    = NewString("__concat")
	metaTostring  = NewString("__tostring")
	metaMetatable = NewString("__metatable")
	metaGc        = NewString("__gc")
)
