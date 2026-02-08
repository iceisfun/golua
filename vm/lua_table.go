package vm

// LuaTable is the interface for Lua table values.
// The concrete *Table type implements this interface. Custom implementations
// may be provided for specialized use cases (e.g., virtual filesystems, proxies).
type LuaTable interface {
	Get(key Value) Value
	Set(key Value, val Value)
	Delete(key Value)
	Next(key Value) (nextKey Value, val Value)
	Len() int
	Metatable() LuaTable
	SetMetatable(mt LuaTable)
}

// Compile-time check that *Table implements LuaTable.
var _ LuaTable = (*Table)(nil)

// Pre-allocated metamethod name values to avoid repeated NewString allocations
// in hot paths like metamethod lookups.
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
