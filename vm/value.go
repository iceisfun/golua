// Package vm implements the Lua 5.4 virtual machine.
//
// The VM executes bytecode produced by the [compiler] package using a
// register-based architecture. Each function invocation gets a stack frame
// with registers addressed by index. The VM supports the full Lua 5.4
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
// Lua 5.4 Reference: §2.1 (values and types), §2.4 (metatables and metamethods),
// §2.5 (garbage collection), §2.6 (coroutines), §2.7 (error handling).
package vm

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

// nativeFuncBox wraps a NativeFunc in a heap-allocated struct so that it
// can be stored as a comparable pointer in Value.ptr and used as a Go map
// key (Go function values are not comparable). Each call to NewNativeFunc
// allocates a distinct box, giving each Value reference identity.
type nativeFuncBox struct {
	fn   NativeFunc
	ptr  uintptr // cached reflect pointer for fmt %p output
	nups int     // number of upvalues (for debug.getinfo "u" flag)
}

var emptyStringSentinel byte

// Value represents a Lua runtime value using a tagged union for efficiency.
// The zero value is nil. Values are compared by value for primitive types
// (nil, bool, int, float, string) and by identity for reference types
// (table, function).
//
// Integer and float are distinct subtypes of "number" following Lua 5.4
// semantics. Arithmetic operations preserve integer type when possible.
type Value struct {
	typ     valueType
	num     float64 // float value, or 1.0/0.0 for bool true/false
	integer int64   // integer value
	ptr     any     // string, *Table, *Closure, or NativeFunc
}

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
		v.num = 1
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
	return Value{typ: typeInt, integer: i}
}

// NewFloat creates a float value.
func NewFloat(f float64) Value {
	return Value{typ: typeFloat, num: f}
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
		fn:  f,
		ptr: reflect.ValueOf(f).Pointer(),
	}}
}

// NewNativeFuncWithNups creates a native function value that reports the
// given number of upvalues via debug.getinfo. This is used for functions
// like coroutine.wrap's iterator which conceptually closes over state.
func NewNativeFuncWithNups(f NativeFunc, nups int) Value {
	return Value{typ: typeNativeFunc, ptr: &nativeFuncBox{
		fn:   f,
		ptr:  reflect.ValueOf(f).Pointer(),
		nups: nups,
	}}
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
	return v.num != 0
}

// AsInt returns the integer value (also works for floats that are whole numbers).
func (v Value) AsInt() int64 {
	if v.typ == typeInt {
		return v.integer
	}
	return int64(v.num)
}

// AsFloat returns the float value.
func (v Value) AsFloat() float64 {
	if v.typ == typeInt {
		return float64(v.integer)
	}
	return v.num
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

// NativeFuncNups returns the number of upvalues for a native function value.
func (v Value) NativeFuncNups() int {
	if v.typ == typeNativeFunc {
		return v.ptr.(*nativeFuncBox).nups
	}
	return 0
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
	s = strings.TrimSpace(s)
	if s == "" {
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
	// Handle signed hex (e.g. -0x1, +0xA.8)
	if len(s) > 3 && (s[0] == '+' || s[0] == '-') &&
		s[1] == '0' && (s[2] == 'x' || s[2] == 'X') {
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
		return float64(v.integer), true
	case typeFloat:
		return v.num, true
	case typeString:
		s := strings.TrimSpace(v.ptr.(string))
		if s == "" {
			return 0, false
		}
		// Reject textual inf/nan tokens
		if rejectInfNan(s) {
			return 0, false
		}
		// Try parsing as float (accept ErrRange for overflow → ±Inf)
		if f, err := strconv.ParseFloat(s, 64); err == nil || errors.Is(err, strconv.ErrRange) {
			return f, true
		}
		// Try parsing hex (0x prefix)
		if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
			if i, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
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
		return v.integer, true
	case typeFloat:
		f := v.num
		i := int64(f)
		if float64(i) == f {
			return i, true
		}
		return 0, false
	case typeString:
		// Try parsing string as number, then convert to int
		s := strings.TrimSpace(v.ptr.(string))
		if s == "" {
			return 0, false
		}
		// Try hex integer first
		if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
			if i, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
				return i, true
			}
			// Try unsigned hex for values like 0xFFFFFFFFFFFFFFFF
			if u, err := strconv.ParseUint(s[2:], 16, 64); err == nil {
				return int64(u), true
			}
		}
		// Try direct decimal integer parse (preserves precision for maxint)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
		// Try as float, then check if it's a whole number
		// Accept ErrRange (overflow → ±Inf) but Inf won't pass the int check below
		if f, err := strconv.ParseFloat(s, 64); err == nil || errors.Is(err, strconv.ErrRange) {
			i := int64(f)
			if float64(i) == f {
				return i, true
			}
		}
		return 0, false
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
		return v.num != 0
	}
	return true
}

// String returns a string representation of the value.
func (v Value) String() string {
	switch v.typ {
	case typeNil:
		return "nil"
	case typeBool:
		if v.num != 0 {
			return "true"
		}
		return "false"
	case typeInt:
		return fmt.Sprintf("%d", v.integer)
	case typeFloat:
		f := v.num
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
		s := fmt.Sprintf("%.14g", f)
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
		return fmt.Sprintf("function: 0x%x", v.ptr.(*nativeFuncBox).ptr)
	case typeUpvalue:
		return fmt.Sprintf("userdata: %p", v.ptr)
	default:
		return "???"
	}
}

// PointerString returns the %p representation for string.format.
// Tables, strings, and functions return a hex address; other types return "(null)".
func (v Value) PointerString() string {
	switch v.typ {
	case typeTable:
		return fmt.Sprintf("%p", v.ptr)
	case typeString:
		s := v.ptr.(string)
		if len(s) == 0 {
			return fmt.Sprintf("%p", &emptyStringSentinel)
		}
		ptr := unsafe.StringData(s)
		if ptr == nil {
			return "(null)"
		}
		return fmt.Sprintf("%p", ptr)
	case typeFunction:
		return fmt.Sprintf("%p", v.ptr)
	case typeNativeFunc:
		return fmt.Sprintf("0x%x", v.ptr.(*nativeFuncBox).ptr)
	default:
		return "(null)"
	}
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
				return intFloatEqual(v.integer, other.num)
			}
			return intFloatEqual(other.integer, v.num)
		}
		return false
	}
	switch v.typ {
	case typeNil:
		return true
	case typeBool:
		return v.num == other.num
	case typeInt:
		return v.integer == other.integer
	case typeFloat:
		return v.num == other.num
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
			return v.integer < other.integer, true
		}
		if v.typ == typeInt {
			return intFloatLessThan(v.integer, other.num), true
		}
		if other.typ == typeInt {
			return floatIntLessThan(v.num, other.integer), true
		}
		return v.num < other.num, true
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
			return v.integer <= other.integer, true
		}
		if v.typ == typeInt {
			// i <= f: NaN must return false
			if math.IsNaN(other.num) {
				return false, true
			}
			return !floatIntLessThan(other.num, v.integer), true
		}
		if other.typ == typeInt {
			// f <= i: NaN must return false
			if math.IsNaN(v.num) {
				return false, true
			}
			return !intFloatLessThan(other.integer, v.num), true
		}
		return v.num <= other.num, true
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
// Lua 5.4 allows hex floats without the mandatory 'p' exponent that Go requires.
func ParseHexFloat(s string) (float64, bool) {
	// Handle optional sign
	sign := 1.0
	body := s
	if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
		if body[0] == '-' {
			sign = -1.0
		}
		body = body[1:]
	}
	// Strip 0x/0X prefix
	if len(body) < 2 || body[0] != '0' || (body[1] != 'x' && body[1] != 'X') {
		return 0, false
	}
	body = body[2:]
	if len(body) == 0 {
		return 0, false
	}

	// Split into integer.fraction and optional pExponent
	var intPart, fracPart string
	var expPart string
	pIdx := strings.IndexAny(body, "pP")
	if pIdx >= 0 {
		expPart = body[pIdx+1:]
		body = body[:pIdx]
	}
	dotIdx := strings.IndexByte(body, '.')
	if dotIdx >= 0 {
		intPart = body[:dotIdx]
		fracPart = body[dotIdx+1:]
	} else {
		intPart = body
	}

	// Must have at least one hex digit somewhere
	if intPart == "" && fracPart == "" {
		return 0, false
	}

	// Parse integer and fractional hex digits, tracking binary exponent
	// separately to handle very long digit strings without overflow.
	var value float64
	binExp := 0
	const maxSigDigits = 15 // enough for float64 precision (60 bits > 53)
	sigDigits := 0
	gotNonZero := false

	for _, c := range intPart {
		d := hexDigit(c)
		if d < 0 {
			return 0, false
		}
		if d != 0 {
			gotNonZero = true
		}
		if gotNonZero {
			if sigDigits < maxSigDigits {
				value = value*16 + float64(d)
				sigDigits++
			} else {
				// Beyond precision: digit contributes to exponent only
				binExp += 4
			}
		} else {
			// Leading zeros don't affect value or exponent
		}
	}

	// Parse fractional part
	if fracPart != "" {
		fracExp := 0
		for _, c := range fracPart {
			d := hexDigit(c)
			if d < 0 {
				return 0, false
			}
			fracExp -= 4
			if d != 0 {
				gotNonZero = true
			}
			if gotNonZero && sigDigits < maxSigDigits {
				value = value*16 + float64(d)
				binExp += fracExp
				fracExp = 0
				sigDigits++
			} else if !gotNonZero {
				// Leading fractional zeros adjust the exponent
				binExp += fracExp
				fracExp = 0
			}
			// Digits beyond precision are dropped
		}
	}

	// Parse binary exponent
	if pIdx >= 0 && expPart == "" {
		return 0, false // 'p'/'P' present but no exponent digits
	}
	if expPart != "" {
		expSign := 1
		if len(expPart) > 0 && (expPart[0] == '+' || expPart[0] == '-') {
			if expPart[0] == '-' {
				expSign = -1
			}
			expPart = expPart[1:]
		}
		if expPart == "" {
			return 0, false
		}
		exp := 0
		for _, c := range expPart {
			if c < '0' || c > '9' {
				return 0, false
			}
			exp = exp*10 + int(c-'0')
			if exp > 100000 {
				// Cap to avoid int overflow; result will be 0 or ±Inf
				exp = 100000
			}
		}
		binExp += expSign * exp
	}

	return sign * math.Ldexp(value, binExp), true
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
