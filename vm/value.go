// Package vm executes Lua bytecode compiled by the compiler package.
package vm

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Value represents a Lua runtime value.
// Uses a tagged union approach for efficiency.
type Value struct {
	typ     valueType
	num     float64 // authoritative for typeFloat, typeBool
	integer int64   // authoritative for typeInt
	ptr     any
}

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
func NewNativeFunc(f NativeFunc) Value {
	return Value{typ: typeNativeFunc, ptr: f}
}

// Type queries
func (v Value) IsNil() bool        { return v.typ == typeNil }
func (v Value) IsBool() bool       { return v.typ == typeBool }
func (v Value) IsInt() bool        { return v.typ == typeInt }
func (v Value) IsFloat() bool      { return v.typ == typeFloat }
func (v Value) IsNumber() bool     { return v.typ == typeInt || v.typ == typeFloat }
func (v Value) IsString() bool     { return v.typ == typeString }
func (v Value) IsTable() bool      { return v.typ == typeTable }
func (v Value) IsFunction() bool   { return v.typ == typeFunction }
func (v Value) IsNativeFunc() bool { return v.typ == typeNativeFunc }
func (v Value) IsCallable() bool   { return v.typ == typeFunction || v.typ == typeNativeFunc }

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
		return v.ptr.(NativeFunc)
	}
	return nil
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
		if i, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
			return NewInt(i), true
		}
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
		// Try hex first
		if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
			if i, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
				return i, true
			}
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
		s := fmt.Sprintf("%.14g", f)
		if !strings.ContainsAny(s, ".eEnNiI") {
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
	default:
		return "???"
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
		return reflect.ValueOf(v.ptr).Pointer() == reflect.ValueOf(other.ptr).Pointer()
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

// LuaError wraps a Lua Value as a Go error so that error() can propagate
// arbitrary Lua values (tables, numbers, etc.) through panic/recover
// and pcall/xpcall can return the original value instead of a string.
type LuaError struct {
	Value Value
}

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

	// Parse integer part
	var value float64
	for _, c := range intPart {
		d := hexDigit(c)
		if d < 0 {
			return 0, false
		}
		value = value*16 + float64(d)
	}

	// Parse fractional part
	if fracPart != "" {
		mult := 1.0 / 16.0
		for _, c := range fracPart {
			d := hexDigit(c)
			if d < 0 {
				return 0, false
			}
			value += float64(d) * mult
			mult /= 16.0
		}
	}

	// Parse binary exponent
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
		}
		value *= math.Pow(2, float64(expSign*exp))
	}

	return sign * value, true
}

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
