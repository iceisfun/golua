package vm

// Metamethod is the metatable key of a Lua metamethod such as "__index",
// including the leading "__" prefix.
//
// It is deliberately a string *alias* (not a defined type) so that metamethod
// constants remain usable anywhere a plain string key is expected — table
// SetString/Get, GetMetafield, callMetamethod, and so on — without explicit
// conversions. The alias exists to give these keys a single canonical
// definition and a self-documenting type in signatures.
type Metamethod = string

// Standard Lua metamethod keys. These are the only strings that should be used
// to reference metamethods anywhere in the interpreter; do not write the raw
// "__x" literals at call sites.
const (
	// Access and call
	MetaIndex    Metamethod = "__index"
	MetaNewIndex Metamethod = "__newindex"
	MetaCall     Metamethod = "__call"

	// Lifecycle and identity
	MetaGC        Metamethod = "__gc"
	MetaClose     Metamethod = "__close"
	MetaMode      Metamethod = "__mode"
	MetaName      Metamethod = "__name"
	MetaMetatable Metamethod = "__metatable"
	MetaTostring  Metamethod = "__tostring"
	MetaPairs     Metamethod = "__pairs"

	// Length, comparison, concatenation
	MetaLen    Metamethod = "__len"
	MetaEq     Metamethod = "__eq"
	MetaLt     Metamethod = "__lt"
	MetaLe     Metamethod = "__le"
	MetaConcat Metamethod = "__concat"

	// Arithmetic
	MetaAdd  Metamethod = "__add"
	MetaSub  Metamethod = "__sub"
	MetaMul  Metamethod = "__mul"
	MetaDiv  Metamethod = "__div"
	MetaIDiv Metamethod = "__idiv"
	MetaMod  Metamethod = "__mod"
	MetaPow  Metamethod = "__pow"
	MetaUnm  Metamethod = "__unm"

	// Bitwise
	MetaBAnd Metamethod = "__band"
	MetaBOr  Metamethod = "__bor"
	MetaBXor Metamethod = "__bxor"
	MetaBNot Metamethod = "__bnot"
	MetaShl  Metamethod = "__shl"
	MetaShr  Metamethod = "__shr"
)

// metamethodPrefix is the leading marker shared by every metamethod key.
const metamethodPrefix = "__"

// MetaEvent returns a metamethod's "event" name: the key without the leading
// "__" prefix (e.g. MetaEvent(MetaAdd) == "add"). This is the form Lua uses in
// runtime error messages and debug "namewhat" reporting.
func MetaEvent(m Metamethod) string {
	return m[len(metamethodPrefix):]
}
