// Package vm implements the Lua 5.5 virtual machine.
//
// The VM executes bytecode produced by the [compiler] package using a
// register-based architecture. Each function invocation gets a stack frame
// with registers addressed by index. The VM supports the full Lua 5.5
// feature set: closures with upvalue capture, coroutines via goroutine-based
// cooperative scheduling, metamethods for all operators, and protected calls
// (pcall/xpcall) via Go's panic/recover.
//
// # Value Representation
//
// Lua values are represented by the [Value] struct, a tagged union that avoids
// interface boxing for common types (nil, bool, int, float). Tables implement
// the [LuaTable] interface, allowing custom virtual table implementations.
//
// # Error Handling
//
// Lua errors are propagated as Go panics wrapping [LuaError]. Protected call
// boundaries (pcall, xpcall) recover these panics and return the error value.
// Go errors from native functions are converted to Lua errors at the boundary.
//
// # Provider Interfaces
//
// The VM uses provider interfaces ([LuaIoProvider], [LuaOsProvider],
// [LuaChanProvider], etc.) to abstract system operations, enabling sandboxed
// execution without direct filesystem or OS access.
//
// # Coroutines
//
// Coroutines run as separate goroutines communicating via channels. Each
// coroutine gets its own [VM] instance sharing globals with the parent.
// Yield/resume synchronize through paired channels.
//
// Lua 5.5 Reference: §2.1 (values and types), §2.4 (metatables and metamethods),
// §2.5 (garbage collection), §2.6 (coroutines), §2.7 (error handling).
package vm

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// nativeFuncBox wraps a NativeFunc in a heap-allocated struct so that it
// can be stored as a comparable pointer in Value.ptr and used as a Go map
// key (Go function values are not comparable). Each call to NewNativeFunc
// allocates a distinct box, giving each Value reference identity.
type nativeFuncBox struct {
	fn         NativeFunc
	nups       int        // number of upvalues (for debug.getinfo "u" flag)
	upvalues   []Value    // upvalue slots for C closure emulation
	upvalueIDs []*Upvalue // identity tokens for debug.upvalueid
}

var emptyStringSentinel byte

type stringPointerToken struct{ _ byte }

// stringInternMap is the per-VM interning cache for short string pointer identity.
// Stored in VM.internalState under the "stringptrs" key.
type stringInternMap struct {
	ids map[string]*stringPointerToken
}

const stringInternsKey = "stringptrs"

// getStringInterns returns the per-VM string interning map, creating it lazily.
func getStringInterns(vm *VM) *stringInternMap {
	if r := vm.InternalState(stringInternsKey); r != nil {
		return r.(*stringInternMap)
	}
	m := &stringInternMap{ids: make(map[string]*stringPointerToken)}
	vm.SetInternalState(stringInternsKey, m)
	return m
}

// stringPointerID returns a stable pointer identity for a short string within
// a VM's lifetime. The interning map is shared between a root VM and its
// coroutines, and is released when the VM is garbage collected.
func (vm *VM) stringPointerID(s string) any {
	if s == "" {
		return &emptyStringSentinel
	}
	m := getStringInterns(vm)
	// The internalMu already serializes lazy-init; after that, only one
	// goroutine at a time executes Lua in a given VM tree (coroutines
	// alternate), so no extra lock is needed on the map itself.
	if id, ok := m.ids[s]; ok {
		return id
	}
	id := &stringPointerToken{}
	m.ids[s] = id
	return id
}

// Value represents a Lua runtime value using a tagged union for efficiency.
// The zero value is nil. Values are compared by value for primitive types
// (nil, bool, int, float, string) and by identity for reference types
// (table, function).
//
// Integer and float are distinct subtypes of "number" following Lua 5.4
// semantics. Arithmetic operations preserve integer type when possible.
type Value struct {
	typ valueType
	n   uint64 // numeric word: float64 bits (float, and 1.0/0.0 for bool), or int64 bits (integer)
	ptr any    // string, *Table, *Closure, or NativeFunc
}

// fval decodes the numeric word as a float64. Meaningful when typ is
// typeFloat (or typeBool, where the word holds the bits of 1.0/0.0).
func (v Value) fval() float64 { return math.Float64frombits(v.n) }

// ival decodes the numeric word as an int64. Meaningful when typ is typeInt.
func (v Value) ival() int64 { return int64(v.n) }

// valueType tags the kind of Lua value stored in a Value struct.
type valueType byte

const (
	typeNil valueType = iota
	typeBool
	typeInt
	typeFloat
	typeString
	typeTable
	typeFunction
	typeNativeFunc
	typeUpvalue
)

// Nil is the nil value singleton.
var Nil = Value{typ: typeNil}

// NewBool creates a boolean value.
func NewBool(b bool) Value {
	v := Value{typ: typeBool}
	if b {
		v.n = math.Float64bits(1)
	}
	return v
}

// True and False are convenience values.
var (
	True  = NewBool(true)
	False = NewBool(false)
)

// NewInt creates an integer value.
func NewInt(i int64) Value {
	return Value{typ: typeInt, n: uint64(i)}
}

// NewFloat creates a float value.
func NewFloat(f float64) Value {
	return Value{typ: typeFloat, n: math.Float64bits(f)}
}

// NewString creates a string value.
func NewString(s string) Value {
	return Value{typ: typeString, ptr: s}
}

// NewTable creates a table value.
func NewTable(t LuaTable) Value {
	return Value{typ: typeTable, ptr: t}
}

// NewFunction creates a closure value.
func NewFunction(f *Closure) Value {
	return Value{typ: typeFunction, ptr: f}
}

// NewNativeFunc creates a native function value.
// The function is wrapped in a nativeFuncBox so that the resulting Value
// can be used as a table key (Go function types are not comparable).
func NewNativeFunc(f NativeFunc) Value {
	return Value{typ: typeNativeFunc, ptr: &nativeFuncBox{
		fn: f,
	}}
}

// NewNativeFuncWithNups creates a native function value that reports the
// given number of upvalues via debug.getinfo. This is used for functions
// like coroutine.wrap's iterator which conceptually closes over state.
// Upvalue slots are allocated and initialized to nil.
func NewNativeFuncWithNups(f NativeFunc, nups int) Value {
	uvs := make([]Value, nups)
	ids := make([]*Upvalue, nups)
	for i := range ids {
		ids[i] = &Upvalue{} // unique identity token per slot
	}
	return Value{typ: typeNativeFunc, ptr: &nativeFuncBox{
		fn:         f,
		nups:       nups,
		upvalues:   uvs,
		upvalueIDs: ids,
	}}
}

// NativeFuncUpvalue returns the value of upvalue at 1-based index i for a
// native function. Returns Nil and false if out of range or not a native func.
func (v Value) NativeFuncUpvalue(i int) (Value, bool) {
	if v.typ == typeNativeFunc {
		box := v.ptr.(*nativeFuncBox)
		if i >= 1 && i <= len(box.upvalues) {
			return box.upvalues[i-1], true
		}
	}
	return Nil, false
}

// SetNativeFuncUpvalue sets the value of upvalue at 1-based index i for a
// native function. Returns false if out of range or not a native func.
func (v Value) SetNativeFuncUpvalue(i int, val Value) bool {
	if v.typ == typeNativeFunc {
		box := v.ptr.(*nativeFuncBox)
		if i >= 1 && i <= len(box.upvalues) {
			box.upvalues[i-1] = val
			return true
		}
	}
	return false
}

// NewUpvalueID creates a lightuserdata value wrapping an upvalue pointer.
// Used by debug.upvalueid to return a unique, comparable identifier.
func NewUpvalueID(uv *Upvalue) Value {
	return Value{typ: typeUpvalue, ptr: uv}
}

// IsNil reports whether v is nil.
func (v Value) IsNil() bool { return v.typ == typeNil }

// IsBool reports whether v is a boolean.
func (v Value) IsBool() bool { return v.typ == typeBool }

// IsInt reports whether v is an integer number.
func (v Value) IsInt() bool { return v.typ == typeInt }

// IsFloat reports whether v is a floating-point number.
func (v Value) IsFloat() bool { return v.typ == typeFloat }

// IsNumber reports whether v is a number (integer or float).
func (v Value) IsNumber() bool { return v.typ == typeInt || v.typ == typeFloat }

// IsString reports whether v is a string.
func (v Value) IsString() bool { return v.typ == typeString }

// IsTable reports whether v is a table.
func (v Value) IsTable() bool { return v.typ == typeTable }

// isThread reports whether v is a thread (coroutine).
func (v Value) isThread() bool {
	if v.typ != typeTable {
		return false
	}
	if tbl, ok := v.ptr.(*Table); ok {
		return tbl.isThread
	}
	return false
}

// IsFunction reports whether v is a Lua closure.
func (v Value) IsFunction() bool { return v.typ == typeFunction }

// IsNativeFunc reports whether v is a Go native function.
func (v Value) IsNativeFunc() bool { return v.typ == typeNativeFunc }

// IsCallable reports whether v can be called (closure or native function).
// Note: tables with a __call metamethod are also callable at runtime but
// return false here since the metamethod lookup happens in the VM.
func (v Value) IsCallable() bool { return v.typ == typeFunction || v.typ == typeNativeFunc }

// Type returns the Lua type name.
func (v Value) Type() string {
	switch v.typ {
	case typeNil:
		return "nil"
	case typeBool:
		return "boolean"
	case typeInt, typeFloat:
		return "number"
	case typeString:
		return "string"
	case typeTable:
		if tbl, ok := v.ptr.(*Table); ok && tbl.IsThread() {
			return "thread"
		}
		return "table"
	case typeFunction, typeNativeFunc:
		return "function"
	case typeUpvalue:
		return "userdata"
	default:
		return "unknown"
	}
}

// Value extractors

// AsBool returns the boolean value.
func (v Value) AsBool() bool {
	switch v.typ {
	case typeBool:
		return v.n != 0
	case typeFloat:
		return v.fval() != 0
	default:
		return false
	}
}

// AsInt returns the integer value (also works for floats that are whole numbers).
func (v Value) AsInt() int64 {
	if v.typ == typeInt {
		return v.ival()
	}
	return int64(v.fval())
}

// AsFloat returns the float value.
func (v Value) AsFloat() float64 {
	if v.typ == typeInt {
		return float64(v.ival())
	}
	return v.fval()
}

// AsString returns the string value.
func (v Value) AsString() string {
	if v.typ == typeString {
		return v.ptr.(string)
	}
	return ""
}

// AsTable returns the table value.
func (v Value) AsTable() LuaTable {
	if v.typ == typeTable {
		return v.ptr.(LuaTable)
	}
	return nil
}

// AsClosure returns the closure value.
func (v Value) AsClosure() *Closure {
	if v.typ == typeFunction {
		return v.ptr.(*Closure)
	}
	return nil
}

// AsNativeFunc returns the native function value.
func (v Value) AsNativeFunc() NativeFunc {
	if v.typ == typeNativeFunc {
		return v.ptr.(*nativeFuncBox).fn
	}
	return nil
}

// NativeFuncUpvalueID returns the identity token for upvalue at 1-based index i
// for a native function. Returns nil if out of range or not a native func.
func (v Value) NativeFuncUpvalueID(i int) *Upvalue {
	if v.typ == typeNativeFunc {
		box := v.ptr.(*nativeFuncBox)
		if i >= 1 && i <= len(box.upvalueIDs) {
			return box.upvalueIDs[i-1]
		}
	}
	return nil
}

// NativeFuncNups returns the number of upvalues for a native function value.
func (v Value) NativeFuncNups() int {
	if v.typ == typeNativeFunc {
		return v.ptr.(*nativeFuncBox).nups
	}
	return 0
}

// TrimASCIISpace strips ONLY the bytes Lua's isspace() recognizes:
// '\t', '\n', '\v', '\f', '\r', and ' ' (0x09–0x0D and 0x20).
//
// This must be used in place of strings.TrimSpace anywhere golua coerces a
// string to a number (tonumber, arithmetic-on-strings, math.* on string
// args, %d/%i format coercion, etc.). Go's strings.TrimSpace calls
// unicode.IsSpace, which strips NBSP (U+00A0), en-space (U+2002),
// ideographic space (U+3000), and other Unicode whitespace code points
// that reference Lua does NOT accept as numeric prefixes/suffixes.
func TrimASCIISpace(s string) string {
	lo, hi := 0, len(s)
	for lo < hi {
		c := s[lo]
		if c != ' ' && c != '\t' && c != '\n' && c != '\v' && c != '\f' && c != '\r' {
			break
		}
		lo++
	}
	for hi > lo {
		c := s[hi-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\v' && c != '\f' && c != '\r' {
			break
		}
		hi--
	}
	return s[lo:hi]
}

// rejectInfNan returns true if the trimmed string is an inf/nan token that
// Go's strconv.ParseFloat would accept but Lua 5.4 must reject.
func rejectInfNan(s string) bool {
	lower := strings.ToLower(s)
	if len(lower) == 0 {
		return false
	}
	bare := lower
	if bare[0] == '+' || bare[0] == '-' {
		bare = bare[1:]
	}
	return strings.HasPrefix(bare, "inf") || strings.HasPrefix(bare, "nan")
}

// StringToNumericValue converts a string to a numeric Value, preserving the
// integer/float distinction based on the string format.
func StringToNumericValue(s string) (Value, bool) {
	s = TrimASCIISpace(s)
	if s == "" {
		return Nil, false
	}
	// Reject Go-style underscore digit separators ("1_0"), which strconv accepts
	// but the Lua numeral grammar does not — string arithmetic/coercion must fail
	// on them, matching tonumber and the lexer.
	if strings.IndexByte(s, '_') >= 0 {
		return Nil, false
	}
	// Try hex integer
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		hex := s[2:]
		// Check if it's a hex float (has '.' or 'p'/'P')
		isHexFloat := false
		for _, c := range hex {
			if c == '.' || c == 'p' || c == 'P' {
				isHexFloat = true
				break
			}
		}
		if !isHexFloat {
			// Hex integer: parse with wrapping on overflow (Lua 5.4 behavior)
			if i, err := strconv.ParseUint(hex, 16, 64); err == nil {
				return NewInt(int64(i)), true
			}
			// Overflows uint64: parse digit by digit with modular wrapping
			var result uint64
			valid := false
			for _, c := range hex {
				var d uint64
				switch {
				case c >= '0' && c <= '9':
					d = uint64(c - '0')
				case c >= 'a' && c <= 'f':
					d = uint64(c-'a') + 10
				case c >= 'A' && c <= 'F':
					d = uint64(c-'A') + 10
				default:
					return Nil, false
				}
				result = result*16 + d
				valid = true
			}
			if valid {
				return NewInt(int64(result)), true
			}
			return Nil, false
		}
		// Hex float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return NewFloat(f), true
		}
		if f, ok := ParseHexFloat(s); ok {
			return NewFloat(f), true
		}
		return Nil, false
	}
	// Handle signed hex (e.g. -0x1, +0xA.8, -0xff)
	if len(s) > 3 && (s[0] == '+' || s[0] == '-') &&
		s[1] == '0' && (s[2] == 'x' || s[2] == 'X') {
		hex := s[3:]
		sign := s[0]
		// Check if it's a hex float (has '.' or 'p'/'P')
		isHexFloat := false
		for _, c := range hex {
			if c == '.' || c == 'p' || c == 'P' {
				isHexFloat = true
				break
			}
		}
		if !isHexFloat {
			// Signed hex integer
			if u, err := strconv.ParseUint(hex, 16, 64); err == nil {
				i := int64(u)
				if sign == '-' {
					i = -i
				}
				return NewInt(i), true
			}
			// Overflows uint64: parse digit by digit with modular wrapping
			var result uint64
			valid := false
			for _, c := range hex {
				var d uint64
				switch {
				case c >= '0' && c <= '9':
					d = uint64(c - '0')
				case c >= 'a' && c <= 'f':
					d = uint64(c-'a') + 10
				case c >= 'A' && c <= 'F':
					d = uint64(c-'A') + 10
				default:
					return Nil, false
				}
				result = result*16 + d
				valid = true
			}
			if valid {
				i := int64(result)
				if sign == '-' {
					i = -i
				}
				return NewInt(i), true
			}
			return Nil, false
		}
		// Signed hex float
		if f, ok := ParseHexFloat(s); ok {
			return NewFloat(f), true
		}
	}
	// Try decimal integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return NewInt(i), true
	}
	// Reject textual inf/nan tokens — Go's strconv.ParseFloat accepts these
	// but Lua 5.4 does not (arithmetic coercion must fail on these strings).
	if rejectInfNan(s) {
		return Nil, false
	}
	// Try float (accept ErrRange for overflow → ±Inf, matching Lua 5.4 coercion)
	if f, err := strconv.ParseFloat(s, 64); err == nil || errors.Is(err, strconv.ErrRange) {
		return NewFloat(f), true
	}
	return Nil, false
}

// ToNumber attempts to convert the value to a number.
// Returns (number, true) on success, (0, false) on failure.
func (v Value) ToNumber() (float64, bool) {
	switch v.typ {
	case typeInt:
		return float64(v.ival()), true
	case typeFloat:
		return v.fval(), true
	case typeString:
		s := TrimASCIISpace(v.ptr.(string))
		if s == "" {
			return 0, false
		}
		// Reject underscore digit separators and inf/nan tokens.
		if strings.IndexByte(s, '_') >= 0 {
			return 0, false
		}
		if rejectInfNan(s) {
			return 0, false
		}
		// Try parsing as float (accept ErrRange for overflow → ±Inf)
		if f, err := strconv.ParseFloat(s, 64); err == nil || errors.Is(err, strconv.ErrRange) {
			return f, true
		}
		// Try parsing hex (0x prefix, including signed +0x/-0x)
		hexStart := 0
		if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
			hexStart = 2
		} else if len(s) > 3 && (s[0] == '+' || s[0] == '-') &&
			s[1] == '0' && (s[2] == 'x' || s[2] == 'X') {
			hexStart = 3
		}
		if hexStart > 0 {
			if i, err := strconv.ParseInt(s[hexStart:], 16, 64); err == nil {
				if hexStart == 3 && s[0] == '-' {
					return float64(-i), true
				}
				return float64(i), true
			}
			if f, ok := ParseHexFloat(s); ok {
				return f, true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

// ToInt attempts to convert the value to an integer.
// Returns (int, true) on success, (0, false) on failure.
func (v Value) ToInt() (int64, bool) {
	switch v.typ {
	case typeInt:
		return v.ival(), true
	case typeFloat:
		f := v.fval()
		i := int64(f)
		if float64(i) == f {
			return i, true
		}
		return 0, false
	case typeString:
		// Convert the string to a numeric Value first, then to an integer. This
		// routes hex integers through StringToNumericValue's digit-by-digit
		// modular wraparound, so a hex literal with more than 16 significant
		// digits (e.g. "0x10000000000000000" == 2^64) coerces to integer 0 as in
		// reference Lua, instead of failing ParseInt/ParseUint and reporting "no
		// integer representation". Also inherits the underscore and
		// inf/nan rejection there.
		nv, ok := StringToNumericValue(v.ptr.(string))
		if !ok {
			return 0, false
		}
		return nv.ToInt()
	default:
		return 0, false
	}
}

// ToBool returns the truthiness of the value.
// In Lua, only nil and false are falsy.
func (v Value) ToBool() bool {
	if v.typ == typeNil {
		return false
	}
	if v.typ == typeBool {
		return v.n != 0
	}
	return true
}

// String returns a string representation of the value.
func (v Value) String() string {
	switch v.typ {
	case typeNil:
		return "nil"
	case typeBool:
		if v.n != 0 {
			return "true"
		}
		return "false"
	case typeInt:
		return fmt.Sprintf("%d", v.ival())
	case typeFloat:
		f := v.fval()
		if math.IsInf(f, 1) {
			return "inf"
		}
		if math.IsInf(f, -1) {
			return "-inf"
		}
		if math.IsNaN(f) {
			if math.Signbit(f) {
				return "-nan"
			}
			return "nan"
		}
		// Lua 5.5 shortest round-trip: first try %.15g, then %.17g if
		// parsing back gives a different value. Append ".0" when the
		// result looks like a plain integer (no '.' or exponent).
		s := fmt.Sprintf("%.15g", f)
		if check, err := strconv.ParseFloat(s, 64); err != nil || check != f {
			s = fmt.Sprintf("%.17g", f)
		}
		if !strings.ContainsAny(s, ".eE") {
			s += ".0"
		}
		return s
	case typeString:
		return v.ptr.(string)
	case typeTable:
		if tbl, ok := v.ptr.(*Table); ok && tbl.IsThread() {
			return fmt.Sprintf("thread: %p", v.ptr)
		}
		return fmt.Sprintf("table: %p", v.ptr)
	case typeFunction:
		return fmt.Sprintf("function: %p", v.ptr)
	case typeNativeFunc:
		return fmt.Sprintf("function: %p", v.ptr)
	case typeUpvalue:
		return fmt.Sprintf("userdata: %p", v.ptr)
	default:
		return "???"
	}
}

// PointerString returns the %p representation for string.format.
// Tables, strings, and functions return a hex address; other types return "(null)".
//
// For short string interning (Lua 5.4 semantics where equal short strings share
// the same %p address), use [VM.PointerString] instead — it requires a VM for
// the per-VM interning cache.
func (v Value) PointerString() string {
	switch v.typ {
	case typeTable:
		return fmt.Sprintf("%p", v.ptr)
	case typeString:
		s := v.ptr.(string)
		return fmt.Sprintf("0x%x", reflect.ValueOf(s).Pointer())
	case typeFunction:
		return fmt.Sprintf("%p", v.ptr)
	case typeNativeFunc:
		return fmt.Sprintf("%p", v.ptr)
	case typeUpvalue:
		return fmt.Sprintf("%p", v.ptr)
	default:
		return "(null)"
	}
}

// PointerString returns the %p representation for a value, with short string
// interning (Lua 5.4 semantics: equal short strings ≤40 chars share the same
// %p address). The interning cache is per-VM and released when the VM is
// garbage collected.
func (vm *VM) PointerString(v Value) string {
	if v.typ == typeString {
		s := v.ptr.(string)
		// Short strings (≤40 chars) are interned in Lua 5.4: equal strings
		// share the same %p address.
		// Long strings use reflect to get the underlying data pointer, matching
		// Lua 5.4's behavior where long strings are not interned.
		if len(s) <= 40 {
			return fmt.Sprintf("%p", vm.stringPointerID(s))
		}
		return fmt.Sprintf("0x%x", reflect.ValueOf(s).Pointer())
	}
	return v.PointerString()
}

// intFloatEqual returns true if int i and float f represent the same value.
// This is exact: returns false if f cannot represent i without precision loss.
func intFloatEqual(i int64, f float64) bool {
	return float64(i) == f && int64(f) == i
}

// intFloatLessThan returns true if int i < float f.
func intFloatLessThan(i int64, f float64) bool {
	if math.IsNaN(f) {
		return false
	}
	if f >= 1<<63 {
		return true // f is above all int64
	}
	if f < -(1 << 63) {
		return false // f is below all int64
	}
	fi := int64(f)
	if float64(fi) == f {
		return i < fi
	}
	// f is not a whole number; fi = trunc(f)
	// if f > fi then i < f iff i <= fi
	// if f < fi then i < f iff i < fi
	if f > float64(fi) {
		return i <= fi
	}
	return i < fi
}

// floatIntLessThan returns true if float f < int i.
func floatIntLessThan(f float64, i int64) bool {
	if math.IsNaN(f) {
		return false
	}
	if f >= 1<<63 {
		return false
	}
	if f < -(1 << 63) {
		return true
	}
	fi := int64(f)
	if float64(fi) == f {
		return fi < i
	}
	if f > float64(fi) {
		return fi < i
	}
	return fi-1 < i
}

// Equal checks Lua equality (==).
func (v Value) Equal(other Value) bool {
	if v.typ != other.typ {
		// Special case: int and float can be equal
		if v.IsNumber() && other.IsNumber() {
			if v.typ == typeInt {
				return intFloatEqual(v.ival(), other.fval())
			}
			return intFloatEqual(other.ival(), v.fval())
		}
		return false
	}
	switch v.typ {
	case typeNil:
		return true
	case typeBool:
		return v.n == other.n
	case typeInt:
		return v.n == other.n
	case typeFloat:
		// Decode before comparing: bit equality differs from float equality
		// for NaN (never equal) and ±0.0 (equal despite distinct bits).
		return v.fval() == other.fval()
	case typeString:
		return v.ptr.(string) == other.ptr.(string)
	case typeTable, typeFunction:
		return v.ptr == other.ptr
	case typeNativeFunc:
		return v.ptr == other.ptr
	case typeUpvalue:
		return v.ptr == other.ptr
	default:
		return false
	}
}

// RawEqual checks raw equality without metamethods.
func (v Value) RawEqual(other Value) bool {
	return v.Equal(other)
}

// LessThan checks if v < other.
// Returns (result, ok) where ok is false if comparison is invalid.
func (v Value) LessThan(other Value) (bool, bool) {
	if v.IsNumber() && other.IsNumber() {
		if v.typ == typeInt && other.typ == typeInt {
			return v.ival() < other.ival(), true
		}
		if v.typ == typeInt {
			return intFloatLessThan(v.ival(), other.fval()), true
		}
		if other.typ == typeInt {
			return floatIntLessThan(v.fval(), other.ival()), true
		}
		return v.fval() < other.fval(), true
	}
	if v.IsString() && other.IsString() {
		return v.AsString() < other.AsString(), true
	}
	return false, false
}

// LessEqual checks if v <= other.
func (v Value) LessEqual(other Value) (bool, bool) {
	if v.IsNumber() && other.IsNumber() {
		if v.typ == typeInt && other.typ == typeInt {
			return v.ival() <= other.ival(), true
		}
		if v.typ == typeInt {
			// i <= f: NaN must return false
			if math.IsNaN(other.fval()) {
				return false, true
			}
			return !floatIntLessThan(other.fval(), v.ival()), true
		}
		if other.typ == typeInt {
			// f <= i: NaN must return false
			if math.IsNaN(v.fval()) {
				return false, true
			}
			return !intFloatLessThan(other.ival(), v.fval()), true
		}
		return v.fval() <= other.fval(), true
	}
	if v.IsString() && other.IsString() {
		return v.AsString() <= other.AsString(), true
	}
	return false, false
}

// LuaError wraps an arbitrary Lua [Value] as a Go error. When Lua code calls
// error(v), the VM panics with a *LuaError containing v. Protected call
// boundaries (pcall, xpcall) recover the panic and extract the original value,
// preserving non-string error objects (tables, numbers, etc.) through the
// error propagation chain.
type LuaError struct {
	Value Value
}

// Error returns the string representation of the wrapped Lua value.
func (e *LuaError) Error() string {
	return ValueToString(e.Value)
}

// ParseHexFloat parses a hex float string like "0x.1", "0xA.8", "0x.1p4".
// Lua allows hex floats without the binary 'p' exponent that Go's ParseFloat
// requires, so a missing exponent is normalized to "p0" and the conversion is
// delegated to strconv.ParseFloat — which is correctly rounded from the exact
// hex value, avoiding the double-rounding/truncation of a manual accumulation. Returns false for anything that is not a valid hex float.
func ParseHexFloat(s string) (float64, bool) {
	body := s
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		body = body[1:]
	}
	if len(body) < 2 || body[0] != '0' || (body[1] != 'x' && body[1] != 'X') {
		return 0, false
	}
	// Reject Go-style underscore digit separators, which strconv accepts but
	// Lua does not.
	if strings.IndexByte(s, '_') >= 0 {
		return 0, false
	}
	t := s
	if strings.IndexAny(s, "pP") < 0 {
		t = s + "p0"
	}
	if f, err := strconv.ParseFloat(t, 64); err == nil || errors.Is(err, strconv.ErrRange) {
		return f, true
	}
	return 0, false
}

// hexDigit returns the numeric value of a hex digit (0-15), or -1 if invalid.
func hexDigit(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}

// ValueToString converts a Value to its string representation.
func ValueToString(val Value) string {
	switch {
	case val.IsNil():
		return "nil"
	case val.IsBool():
		if val.AsBool() {
			return "true"
		}
		return "false"
	case val.IsInt():
		return fmt.Sprintf("%d", val.AsInt())
	case val.IsFloat():
		return fmt.Sprintf("%g", val.AsFloat())
	case val.IsString():
		return val.AsString()
	default:
		return fmt.Sprintf("%s: %p", val.Type(), val.ptr)
	}
}
