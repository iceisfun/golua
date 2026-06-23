package vm

// LuaTable is the interface for Lua table values. The concrete [Table] type
// implements this interface. Custom implementations may be provided for
// specialized use cases (virtual filesystems, proxy objects, etc.).
//
// All methods follow Lua 5.5 raw access semantics — metamethods are NOT
// invoked by Get/Set. The VM handles metamethod dispatch separately.
//
// Lua 5.5 Reference: §2.1 (tables), §6.7 (table library).
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
	// IsThread returns whether this table represents a coroutine thread.
	IsThread() bool
	// VMRef returns the coroutine VM reference, or nil if not set.
	VMRef() *VM
}

// Compile-time check that *Table implements LuaTable.
var _ LuaTable = (*Table)(nil)

// Pre-allocated metamethod name values. These are created once at init time
// to avoid repeated string allocation in hot paths like metamethod lookups.
// The full set covers all Lua 5.4 metamethods (§2.4).
var (
	metaIndex     = NewString(MetaIndex)
	metaNewIndex  = NewString(MetaNewIndex)
	metaClose     = NewString(MetaClose)
	metaCall      = NewString(MetaCall)
	metaAdd       = NewString(MetaAdd)
	metaSub       = NewString(MetaSub)
	metaMul       = NewString(MetaMul)
	metaDiv       = NewString(MetaDiv)
	metaMod       = NewString(MetaMod)
	metaPow       = NewString(MetaPow)
	metaUnm       = NewString(MetaUnm)
	metaIDiv      = NewString(MetaIDiv)
	metaBand      = NewString(MetaBAnd)
	metaBor       = NewString(MetaBOr)
	metaBxor      = NewString(MetaBXor)
	metaBnot      = NewString(MetaBNot)
	metaShl       = NewString(MetaShl)
	metaShr       = NewString(MetaShr)
	metaEq        = NewString(MetaEq)
	metaLt        = NewString(MetaLt)
	metaLe        = NewString(MetaLe)
	metaLen       = NewString(MetaLen)
	metaConcat    = NewString(MetaConcat)
	metaTostring  = NewString(MetaTostring)
	metaMetatable = NewString(MetaMetatable)
	metaGc        = NewString(MetaGC)
	metaMode      = NewString(MetaMode)
)
