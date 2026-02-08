package vm

import (
	"math"
	"strings"
	"testing"
)

// --- Constructors ---

func TestNewNil(t *testing.T) {
	v := Nil
	if !v.IsNil() {
		t.Error("Nil should be nil")
	}
	if v.Type() != "nil" {
		t.Errorf("expected type 'nil', got %q", v.Type())
	}
}

func TestNewBoolTrue(t *testing.T) {
	v := NewBool(true)
	if !v.IsBool() {
		t.Error("should be bool")
	}
	if !v.AsBool() {
		t.Error("should be true")
	}
}

func TestNewBoolFalse(t *testing.T) {
	v := NewBool(false)
	if !v.IsBool() {
		t.Error("should be bool")
	}
	if v.AsBool() {
		t.Error("should be false")
	}
}

func TestTrueFalseSingletons(t *testing.T) {
	if !True.AsBool() {
		t.Error("True should be true")
	}
	if False.AsBool() {
		t.Error("False should be false")
	}
}

func TestNewInt(t *testing.T) {
	v := NewInt(42)
	if !v.IsInt() {
		t.Error("should be int")
	}
	if v.AsInt() != 42 {
		t.Errorf("expected 42, got %d", v.AsInt())
	}
}

func TestNewIntNegative(t *testing.T) {
	v := NewInt(-100)
	if v.AsInt() != -100 {
		t.Errorf("expected -100, got %d", v.AsInt())
	}
}

func TestNewIntZero(t *testing.T) {
	v := NewInt(0)
	if v.AsInt() != 0 {
		t.Errorf("expected 0, got %d", v.AsInt())
	}
}

func TestNewIntMaxMin(t *testing.T) {
	max := NewInt(math.MaxInt32)
	if max.AsInt() != math.MaxInt32 {
		t.Errorf("expected MaxInt32, got %d", max.AsInt())
	}
	min := NewInt(math.MinInt32)
	if min.AsInt() != math.MinInt32 {
		t.Errorf("expected MinInt32, got %d", min.AsInt())
	}
}

func TestNewFloat(t *testing.T) {
	v := NewFloat(3.14)
	if !v.IsFloat() {
		t.Error("should be float")
	}
	if math.Abs(v.AsFloat()-3.14) > 1e-10 {
		t.Errorf("expected 3.14, got %f", v.AsFloat())
	}
}

func TestNewFloatZero(t *testing.T) {
	v := NewFloat(0.0)
	if v.AsFloat() != 0.0 {
		t.Errorf("expected 0.0, got %f", v.AsFloat())
	}
}

func TestNewFloatInfinity(t *testing.T) {
	v := NewFloat(math.Inf(1))
	if !math.IsInf(v.AsFloat(), 1) {
		t.Error("expected +Inf")
	}
	v2 := NewFloat(math.Inf(-1))
	if !math.IsInf(v2.AsFloat(), -1) {
		t.Error("expected -Inf")
	}
}

func TestNewFloatNaN(t *testing.T) {
	v := NewFloat(math.NaN())
	if !math.IsNaN(v.AsFloat()) {
		t.Error("expected NaN")
	}
}

func TestNewString(t *testing.T) {
	v := NewString("hello")
	if !v.IsString() {
		t.Error("should be string")
	}
	if v.AsString() != "hello" {
		t.Errorf("expected 'hello', got %q", v.AsString())
	}
}

func TestNewStringEmpty(t *testing.T) {
	v := NewString("")
	if v.AsString() != "" {
		t.Errorf("expected empty string, got %q", v.AsString())
	}
}

func TestNewStringUnicode(t *testing.T) {
	v := NewString("日本語")
	if v.AsString() != "日本語" {
		t.Errorf("expected '日本語', got %q", v.AsString())
	}
}

func TestNewTableValue(t *testing.T) {
	tbl := NewEmptyTable()
	v := NewTable(tbl)
	if !v.IsTable() {
		t.Error("should be table")
	}
	if v.AsTable() == nil {
		t.Error("AsTable should not be nil")
	}
}

func TestNewFunctionValue(t *testing.T) {
	// Minimal closure — no proto needed for type testing
	c := &Closure{}
	v := NewFunction(c)
	if !v.IsFunction() {
		t.Error("should be function")
	}
	if v.AsClosure() != c {
		t.Error("AsClosure should return same pointer")
	}
}

func TestNewNativeFuncValue(t *testing.T) {
	fn := NativeFunc(func(vm *VM) int { return 0 })
	v := NewNativeFunc(fn)
	if !v.IsNativeFunc() {
		t.Error("should be native func")
	}
	if v.AsNativeFunc() == nil {
		t.Error("AsNativeFunc should not be nil")
	}
}

// --- Type queries ---

func TestIsNumber(t *testing.T) {
	if !NewInt(1).IsNumber() {
		t.Error("int should be number")
	}
	if !NewFloat(1.0).IsNumber() {
		t.Error("float should be number")
	}
	if NewString("1").IsNumber() {
		t.Error("string should not be number")
	}
	if Nil.IsNumber() {
		t.Error("nil should not be number")
	}
}

func TestIsCallable(t *testing.T) {
	c := &Closure{}
	fn := NativeFunc(func(vm *VM) int { return 0 })

	if !NewFunction(c).IsCallable() {
		t.Error("function should be callable")
	}
	if !NewNativeFunc(fn).IsCallable() {
		t.Error("native func should be callable")
	}
	if NewInt(1).IsCallable() {
		t.Error("int should not be callable")
	}
	if NewTable(NewEmptyTable()).IsCallable() {
		t.Error("table should not be callable")
	}
}

func TestTypeExclusion(t *testing.T) {
	// Each value should report exactly one primary type
	values := []struct {
		v    Value
		name string
	}{
		{Nil, "nil"},
		{True, "bool"},
		{NewInt(1), "int"},
		{NewFloat(1.0), "float"},
		{NewString("x"), "string"},
		{NewTable(NewEmptyTable()), "table"},
	}

	for _, tc := range values {
		checks := map[string]bool{
			"nil":    tc.v.IsNil(),
			"bool":   tc.v.IsBool(),
			"int":    tc.v.IsInt(),
			"float":  tc.v.IsFloat(),
			"string": tc.v.IsString(),
			"table":  tc.v.IsTable(),
		}
		for name, result := range checks {
			if name == tc.name && !result {
				t.Errorf("%s.Is%s() should be true", tc.name, name)
			} else if name != tc.name && result {
				t.Errorf("%s.Is%s() should be false", tc.name, name)
			}
		}
	}
}

// --- Type() string ---

func TestTypeName(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{Nil, "nil"},
		{True, "boolean"},
		{False, "boolean"},
		{NewInt(0), "number"},
		{NewFloat(0), "number"},
		{NewString(""), "string"},
		{NewTable(NewEmptyTable()), "table"},
		{NewFunction(&Closure{}), "function"},
		{NewNativeFunc(func(vm *VM) int { return 0 }), "function"},
	}
	for _, tc := range tests {
		if tc.v.Type() != tc.want {
			t.Errorf("%v.Type() = %q, want %q", tc.v, tc.v.Type(), tc.want)
		}
	}
}

// --- Extractors on wrong type ---

func TestAsStringOnNonString(t *testing.T) {
	if NewInt(42).AsString() != "" {
		t.Error("AsString on int should return empty string")
	}
	if Nil.AsString() != "" {
		t.Error("AsString on nil should return empty string")
	}
}

func TestAsTableOnNonTable(t *testing.T) {
	if NewInt(42).AsTable() != nil {
		t.Error("AsTable on int should return nil")
	}
	if NewString("x").AsTable() != nil {
		t.Error("AsTable on string should return nil")
	}
}

func TestAsClosureOnNonFunction(t *testing.T) {
	if NewInt(42).AsClosure() != nil {
		t.Error("AsClosure on int should return nil")
	}
	if NewNativeFunc(func(vm *VM) int { return 0 }).AsClosure() != nil {
		t.Error("AsClosure on native func should return nil")
	}
}

func TestAsNativeFuncOnNonNative(t *testing.T) {
	if NewInt(42).AsNativeFunc() != nil {
		t.Error("AsNativeFunc on int should return nil")
	}
	if NewFunction(&Closure{}).AsNativeFunc() != nil {
		t.Error("AsNativeFunc on closure should return nil")
	}
}

// --- ToBool (Lua truthiness) ---

func TestToBool(t *testing.T) {
	tests := []struct {
		v    Value
		want bool
		desc string
	}{
		{Nil, false, "nil is falsy"},
		{False, false, "false is falsy"},
		{True, true, "true is truthy"},
		{NewInt(0), true, "0 is truthy in Lua"},
		{NewInt(1), true, "1 is truthy"},
		{NewFloat(0.0), true, "0.0 is truthy in Lua"},
		{NewString(""), true, "empty string is truthy in Lua"},
		{NewString("x"), true, "non-empty string is truthy"},
		{NewTable(NewEmptyTable()), true, "table is truthy"},
	}
	for _, tc := range tests {
		if tc.v.ToBool() != tc.want {
			t.Errorf("%s: got %v, want %v", tc.desc, tc.v.ToBool(), tc.want)
		}
	}
}

// --- ToNumber ---

func TestToNumberFromInt(t *testing.T) {
	n, ok := NewInt(42).ToNumber()
	if !ok || n != 42 {
		t.Errorf("expected (42, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromFloat(t *testing.T) {
	n, ok := NewFloat(3.14).ToNumber()
	if !ok || math.Abs(n-3.14) > 1e-10 {
		t.Errorf("expected (3.14, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromStringDecimal(t *testing.T) {
	n, ok := NewString("42.5").ToNumber()
	if !ok || n != 42.5 {
		t.Errorf("expected (42.5, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromStringInt(t *testing.T) {
	n, ok := NewString("100").ToNumber()
	if !ok || n != 100 {
		t.Errorf("expected (100, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromStringHex(t *testing.T) {
	n, ok := NewString("0xFF").ToNumber()
	if !ok || n != 255 {
		t.Errorf("expected (255, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromStringHexUpper(t *testing.T) {
	n, ok := NewString("0XAB").ToNumber()
	if !ok || n != 171 {
		t.Errorf("expected (171, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromStringWithWhitespace(t *testing.T) {
	n, ok := NewString("  42  ").ToNumber()
	if !ok || n != 42 {
		t.Errorf("expected (42, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromEmptyString(t *testing.T) {
	_, ok := NewString("").ToNumber()
	if ok {
		t.Error("empty string should not convert to number")
	}
}

func TestToNumberFromNonNumericString(t *testing.T) {
	_, ok := NewString("abc").ToNumber()
	if ok {
		t.Error("non-numeric string should not convert to number")
	}
}

func TestToNumberFromNil(t *testing.T) {
	_, ok := Nil.ToNumber()
	if ok {
		t.Error("nil should not convert to number")
	}
}

func TestToNumberFromBool(t *testing.T) {
	_, ok := True.ToNumber()
	if ok {
		t.Error("bool should not convert to number")
	}
}

func TestToNumberFromTable(t *testing.T) {
	_, ok := NewTable(NewEmptyTable()).ToNumber()
	if ok {
		t.Error("table should not convert to number")
	}
}

// --- ToInt ---

func TestToIntFromInt(t *testing.T) {
	i, ok := NewInt(42).ToInt()
	if !ok || i != 42 {
		t.Errorf("expected (42, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromWholeFloat(t *testing.T) {
	i, ok := NewFloat(10.0).ToInt()
	if !ok || i != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromFractionalFloat(t *testing.T) {
	_, ok := NewFloat(3.14).ToInt()
	if ok {
		t.Error("fractional float should not convert to int")
	}
}

func TestToIntFromStringInt(t *testing.T) {
	i, ok := NewString("99").ToInt()
	if !ok || i != 99 {
		t.Errorf("expected (99, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromStringFloat(t *testing.T) {
	i, ok := NewString("10.0").ToInt()
	if !ok || i != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromStringFractional(t *testing.T) {
	_, ok := NewString("3.14").ToInt()
	if ok {
		t.Error("fractional string should not convert to int")
	}
}

func TestToIntFromStringHex(t *testing.T) {
	i, ok := NewString("0x1F").ToInt()
	if !ok || i != 31 {
		t.Errorf("expected (31, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromNil(t *testing.T) {
	_, ok := Nil.ToInt()
	if ok {
		t.Error("nil should not convert to int")
	}
}

func TestToIntFromBool(t *testing.T) {
	_, ok := True.ToInt()
	if ok {
		t.Error("bool should not convert to int")
	}
}

func TestToIntFromEmptyString(t *testing.T) {
	_, ok := NewString("").ToInt()
	if ok {
		t.Error("empty string should not convert to int")
	}
}

// --- String() representation ---

func TestStringRepresentation(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{Nil, "nil"},
		{True, "true"},
		{False, "false"},
		{NewInt(42), "42"},
		{NewInt(-1), "-1"},
		{NewInt(0), "0"},
		{NewFloat(3.14), "3.14"},
		{NewFloat(1.0), "1.0"},        // whole float gets .1f format
		{NewFloat(0.0), "0.0"},        // zero float
		{NewString("hello"), "hello"},
		{NewString(""), ""},
	}
	for _, tc := range tests {
		if tc.v.String() != tc.want {
			t.Errorf("String(%v) = %q, want %q", tc.v, tc.v.String(), tc.want)
		}
	}
}

func TestStringInfinity(t *testing.T) {
	s := NewFloat(math.Inf(1)).String()
	if s != "+Inf" {
		t.Errorf("expected '+Inf', got %q", s)
	}
}

func TestStringNegativeInfinity(t *testing.T) {
	s := NewFloat(math.Inf(-1)).String()
	if s != "-Inf" {
		t.Errorf("expected '-Inf', got %q", s)
	}
}

func TestStringNaN(t *testing.T) {
	s := NewFloat(math.NaN()).String()
	if s != "NaN" {
		t.Errorf("expected 'NaN', got %q", s)
	}
}

func TestStringTable(t *testing.T) {
	v := NewTable(NewEmptyTable())
	s := v.String()
	if !strings.HasPrefix(s, "table: 0x") {
		t.Errorf("expected 'table: 0x...', got %q", s)
	}
}

func TestStringFunction(t *testing.T) {
	v := NewFunction(&Closure{})
	s := v.String()
	if !strings.HasPrefix(s, "function: 0x") {
		t.Errorf("expected 'function: 0x...', got %q", s)
	}
}

func TestStringNativeFunc(t *testing.T) {
	v := NewNativeFunc(func(vm *VM) int { return 0 })
	s := v.String()
	if !strings.HasPrefix(s, "function: 0x") {
		t.Errorf("expected 'function: 0x...', got %q", s)
	}
}

func TestStringLargeFloat(t *testing.T) {
	// Floats beyond 1e14 use %g format
	s := NewFloat(1e15).String()
	if s != "1e+15" {
		t.Errorf("expected '1e+15', got %q", s)
	}
}

// --- Equal ---

func TestEqualNilNil(t *testing.T) {
	if !Nil.Equal(Nil) {
		t.Error("nil == nil should be true")
	}
}

func TestEqualBoolSame(t *testing.T) {
	if !True.Equal(True) {
		t.Error("true == true should be true")
	}
	if !False.Equal(False) {
		t.Error("false == false should be true")
	}
}

func TestEqualBoolDifferent(t *testing.T) {
	if True.Equal(False) {
		t.Error("true == false should be false")
	}
}

func TestEqualIntSame(t *testing.T) {
	if !NewInt(42).Equal(NewInt(42)) {
		t.Error("42 == 42 should be true")
	}
}

func TestEqualIntDifferent(t *testing.T) {
	if NewInt(1).Equal(NewInt(2)) {
		t.Error("1 == 2 should be false")
	}
}

func TestEqualIntFloat(t *testing.T) {
	// Lua: int and float with same numeric value are equal
	if !NewInt(42).Equal(NewFloat(42.0)) {
		t.Error("42 == 42.0 should be true")
	}
	if NewInt(42).Equal(NewFloat(42.5)) {
		t.Error("42 == 42.5 should be false")
	}
}

func TestEqualFloatSame(t *testing.T) {
	if !NewFloat(3.14).Equal(NewFloat(3.14)) {
		t.Error("3.14 == 3.14 should be true")
	}
}

func TestEqualStringSame(t *testing.T) {
	if !NewString("abc").Equal(NewString("abc")) {
		t.Error("'abc' == 'abc' should be true")
	}
}

func TestEqualStringDifferent(t *testing.T) {
	if NewString("abc").Equal(NewString("def")) {
		t.Error("'abc' == 'def' should be false")
	}
}

func TestEqualTableIdentity(t *testing.T) {
	tbl := NewEmptyTable()
	v1 := NewTable(tbl)
	v2 := NewTable(tbl)
	if !v1.Equal(v2) {
		t.Error("same table should be equal")
	}
}

func TestEqualTableDifferent(t *testing.T) {
	v1 := NewTable(NewEmptyTable())
	v2 := NewTable(NewEmptyTable())
	if v1.Equal(v2) {
		t.Error("different tables should not be equal")
	}
}

func TestEqualFunctionIdentity(t *testing.T) {
	c := &Closure{}
	v1 := NewFunction(c)
	v2 := NewFunction(c)
	if !v1.Equal(v2) {
		t.Error("same closure should be equal")
	}
}

func TestEqualCrossType(t *testing.T) {
	// Different types (non-number) are never equal
	if Nil.Equal(False) {
		t.Error("nil should not equal false")
	}
	if NewInt(0).Equal(False) {
		t.Error("0 should not equal false")
	}
	if NewString("1").Equal(NewInt(1)) {
		t.Error("string '1' should not equal int 1")
	}
	if Nil.Equal(NewString("")) {
		t.Error("nil should not equal empty string")
	}
	if Nil.Equal(NewInt(0)) {
		t.Error("nil should not equal 0")
	}
}

func TestEqualNaNNotEqualToItself(t *testing.T) {
	nan := NewFloat(math.NaN())
	if nan.Equal(nan) {
		t.Error("NaN should not equal itself")
	}
}

// --- RawEqual ---

func TestRawEqual(t *testing.T) {
	// RawEqual currently delegates to Equal
	if !NewInt(1).RawEqual(NewInt(1)) {
		t.Error("rawequal(1, 1) should be true")
	}
	if NewInt(1).RawEqual(NewInt(2)) {
		t.Error("rawequal(1, 2) should be false")
	}
}

// --- LessThan ---

func TestLessThanNumbers(t *testing.T) {
	lt, ok := NewInt(1).LessThan(NewInt(2))
	if !ok || !lt {
		t.Error("1 < 2 should be true")
	}
	lt, ok = NewInt(2).LessThan(NewInt(1))
	if !ok || lt {
		t.Error("2 < 1 should be false")
	}
	lt, ok = NewInt(1).LessThan(NewInt(1))
	if !ok || lt {
		t.Error("1 < 1 should be false")
	}
}

func TestLessThanIntFloat(t *testing.T) {
	lt, ok := NewInt(1).LessThan(NewFloat(1.5))
	if !ok || !lt {
		t.Error("1 < 1.5 should be true")
	}
}

func TestLessThanStrings(t *testing.T) {
	lt, ok := NewString("a").LessThan(NewString("b"))
	if !ok || !lt {
		t.Error("'a' < 'b' should be true")
	}
	lt, ok = NewString("b").LessThan(NewString("a"))
	if !ok || lt {
		t.Error("'b' < 'a' should be false")
	}
}

func TestLessThanIncomparable(t *testing.T) {
	_, ok := NewInt(1).LessThan(NewString("1"))
	if ok {
		t.Error("int < string should not be valid")
	}
	_, ok = Nil.LessThan(Nil)
	if ok {
		t.Error("nil < nil should not be valid")
	}
}

// --- LessEqual ---

func TestLessEqualNumbers(t *testing.T) {
	le, ok := NewInt(1).LessEqual(NewInt(1))
	if !ok || !le {
		t.Error("1 <= 1 should be true")
	}
	le, ok = NewInt(1).LessEqual(NewInt(2))
	if !ok || !le {
		t.Error("1 <= 2 should be true")
	}
	le, ok = NewInt(2).LessEqual(NewInt(1))
	if !ok || le {
		t.Error("2 <= 1 should be false")
	}
}

func TestLessEqualStrings(t *testing.T) {
	le, ok := NewString("abc").LessEqual(NewString("abc"))
	if !ok || !le {
		t.Error("'abc' <= 'abc' should be true")
	}
	le, ok = NewString("abc").LessEqual(NewString("abd"))
	if !ok || !le {
		t.Error("'abc' <= 'abd' should be true")
	}
}

func TestLessEqualIncomparable(t *testing.T) {
	_, ok := NewInt(1).LessEqual(NewString("1"))
	if ok {
		t.Error("int <= string should not be valid")
	}
}

// --- Edge cases ---

func TestIntAsFloat(t *testing.T) {
	// AsFloat on an int returns the float64 value
	v := NewInt(42)
	if v.AsFloat() != 42.0 {
		t.Errorf("int AsFloat: expected 42.0, got %f", v.AsFloat())
	}
}

func TestFloatAsInt(t *testing.T) {
	// AsInt on a float truncates
	v := NewFloat(3.9)
	if v.AsInt() != 3 {
		t.Errorf("float AsInt: expected 3, got %d", v.AsInt())
	}
}

func TestNilAsBool(t *testing.T) {
	// AsBool checks the num field directly; for nil, num is 0
	if Nil.AsBool() {
		t.Error("nil AsBool should be false")
	}
}

func TestToNumberFromNegativeString(t *testing.T) {
	n, ok := NewString("-42.5").ToNumber()
	if !ok || n != -42.5 {
		t.Errorf("expected (-42.5, true), got (%f, %v)", n, ok)
	}
}

func TestToNumberFromScientificNotation(t *testing.T) {
	n, ok := NewString("1e10").ToNumber()
	if !ok || n != 1e10 {
		t.Errorf("expected (1e10, true), got (%f, %v)", n, ok)
	}
}

func TestToIntFromNegativeInt(t *testing.T) {
	i, ok := NewInt(-42).ToInt()
	if !ok || i != -42 {
		t.Errorf("expected (-42, true), got (%d, %v)", i, ok)
	}
}

func TestToIntFromNegativeFloat(t *testing.T) {
	i, ok := NewFloat(-10.0).ToInt()
	if !ok || i != -10 {
		t.Errorf("expected (-10, true), got (%d, %v)", i, ok)
	}
}

// --- int64 precision tests ---

func TestIntMaxInt64Precision(t *testing.T) {
	max := NewInt(math.MaxInt64)
	if max.AsInt() != math.MaxInt64 {
		t.Errorf("expected MaxInt64 (%d), got %d", int64(math.MaxInt64), max.AsInt())
	}
	min := NewInt(math.MinInt64)
	if min.AsInt() != math.MinInt64 {
		t.Errorf("expected MinInt64 (%d), got %d", int64(math.MinInt64), min.AsInt())
	}
	// String representation must be exact
	if max.String() != "9223372036854775807" {
		t.Errorf("MaxInt64 string: expected '9223372036854775807', got %q", max.String())
	}
	if min.String() != "-9223372036854775808" {
		t.Errorf("MinInt64 string: expected '-9223372036854775808', got %q", min.String())
	}
}

func TestIntMaxInt64RoundTrip(t *testing.T) {
	// ToInt on a NewInt should round-trip exactly
	v := NewInt(math.MaxInt64)
	i, ok := v.ToInt()
	if !ok || i != math.MaxInt64 {
		t.Errorf("ToInt round-trip failed: got (%d, %v)", i, ok)
	}
}

func TestIntFloatEqualityEdgeCases(t *testing.T) {
	// MaxInt64 cannot be exactly represented as float64
	maxI := NewInt(math.MaxInt64)
	maxF := NewFloat(float64(math.MaxInt64))
	if maxI.Equal(maxF) {
		t.Error("MaxInt64 should NOT equal float64(MaxInt64) because float64 rounds it")
	}

	// Small integers should be equal across types
	if !NewInt(42).Equal(NewFloat(42.0)) {
		t.Error("42 == 42.0 should be true")
	}

	// 2^53 fits exactly in float64
	big := int64(1) << 53
	if !NewInt(big).Equal(NewFloat(float64(big))) {
		t.Error("2^53 should equal float64(2^53)")
	}

	// 2^53 + 1 does NOT fit exactly in float64
	big1 := (int64(1) << 53) + 1
	if NewInt(big1).Equal(NewFloat(float64(big1))) {
		t.Error("2^53+1 should NOT equal float64(2^53+1)")
	}
}

func TestIntComparisonEdgeCases(t *testing.T) {
	// int vs float comparison near boundaries
	maxI := NewInt(math.MaxInt64)
	maxF := NewFloat(float64(math.MaxInt64))

	// float64(MaxInt64) rounds up to 2^63, so MaxInt64 < float64(MaxInt64)
	lt, ok := maxI.LessThan(maxF)
	if !ok || !lt {
		t.Error("MaxInt64 should be < float64(MaxInt64) because float rounds up")
	}

	// Negative: MinInt64 vs its float
	minI := NewInt(math.MinInt64)
	minF := NewFloat(float64(math.MinInt64))
	// float64(MinInt64) is exact (-2^63 is a power of 2)
	if !minI.Equal(minF) {
		t.Error("MinInt64 should equal float64(MinInt64) because -2^63 is exact in float64")
	}

	// int-int comparison
	lt, ok = NewInt(math.MinInt64).LessThan(NewInt(math.MaxInt64))
	if !ok || !lt {
		t.Error("MinInt64 < MaxInt64 should be true")
	}

	// NaN comparison
	lt, ok = NewInt(0).LessThan(NewFloat(math.NaN()))
	if !ok || lt {
		t.Error("0 < NaN should be false")
	}
	le, ok := NewInt(0).LessEqual(NewFloat(math.NaN()))
	if !ok || le {
		t.Error("0 <= NaN should be false")
	}
}
