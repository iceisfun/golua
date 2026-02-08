// Package vm executes Lua bytecode compiled by the compiler package.
package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Value represents a Lua runtime value.
// Uses a tagged union approach for efficiency.
type Value struct {
	typ valueType
	// For numbers, we store both representations and use typ to know which is authoritative
	num float64
	// For non-numeric types
	ptr any
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
	return Value{typ: typeInt, num: float64(i)}
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
	return int64(v.num)
}

// AsFloat returns the float value.
func (v Value) AsFloat() float64 {
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

// ToNumber attempts to convert the value to a number.
// Returns (number, true) on success, (0, false) on failure.
func (v Value) ToNumber() (float64, bool) {
	switch v.typ {
	case typeInt, typeFloat:
		return v.num, true
	case typeString:
		s := strings.TrimSpace(v.ptr.(string))
		if s == "" {
			return 0, false
		}
		// Try parsing as float (which also handles integers)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		// Try parsing hex (0x prefix)
		if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
			if i, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
				return float64(i), true
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
		return int64(v.num), true
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
		if f, err := strconv.ParseFloat(s, 64); err == nil {
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
		return fmt.Sprintf("%d", int64(v.num))
	case typeFloat:
		f := v.num
		if f == math.Trunc(f) && !math.IsInf(f, 0) && math.Abs(f) < 1e14 {
			return fmt.Sprintf("%.1f", f)
		}
		return fmt.Sprintf("%g", f)
	case typeString:
		return v.ptr.(string)
	case typeTable:
		return fmt.Sprintf("table: %p", v.ptr)
	case typeFunction:
		return fmt.Sprintf("function: %p", v.ptr)
	case typeNativeFunc:
		return fmt.Sprintf("function: %p", v.ptr)
	default:
		return "???"
	}
}

// Equal checks Lua equality (==).
func (v Value) Equal(other Value) bool {
	if v.typ != other.typ {
		// Special case: int and float can be equal
		if v.IsNumber() && other.IsNumber() {
			return v.num == other.num
		}
		return false
	}
	switch v.typ {
	case typeNil:
		return true
	case typeBool, typeInt, typeFloat:
		return v.num == other.num
	case typeString:
		return v.ptr.(string) == other.ptr.(string)
	case typeTable, typeFunction, typeNativeFunc:
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
		return v.num <= other.num, true
	}
	if v.IsString() && other.IsString() {
		return v.AsString() <= other.AsString(), true
	}
	return false, false
}
