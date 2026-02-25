package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// helper: run Lua code and return results + captured output + error
func runLua(t *testing.T, code string) ([]vm.Value, []string, error) {
	t.Helper()
	block, err := parser.Parse("test", code)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)

	var results []vm.Value
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case *vm.LuaError:
					runErr = e
				case error:
					runErr = e
				case string:
					runErr = fmt.Errorf("%s", e)
				default:
					runErr = fmt.Errorf("%v", r)
				}
			}
		}()
		results, runErr = v.Run(proto)
	}()

	return results, v.OutputLines(), runErr
}

// helper: run Lua and expect success, return results
func mustRun(t *testing.T, code string) []vm.Value {
	t.Helper()
	results, _, err := runLua(t, code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	return results
}

// ============================================================
// BUG 1: string.reverse reverses runes instead of bytes
// Lua strings are byte sequences. string.reverse should reverse bytes.
// ============================================================
func TestBug_StringReverse_ByteVsRune(t *testing.T) {
	// "é" is U+00E9, encoded as bytes 0xC3 0xA9 in UTF-8
	// string.reverse should give 0xA9 0xC3 (byte reversal)
	results := mustRun(t, `
		local s = "\xC3\xA9"  -- "é" in UTF-8
		local r = string.reverse(s)
		return string.byte(r, 1), string.byte(r, 2)
	`)

	if len(results) < 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	b1 := results[0].AsInt()
	b2 := results[1].AsInt()

	if b1 != 0xA9 || b2 != 0xC3 {
		t.Errorf("string.reverse reverses runes instead of bytes")
		t.Errorf("  Expected bytes: [0xA9, 0xC3]")
		t.Errorf("  Got bytes:      [0x%02X, 0x%02X]", b1, b2)
		t.Errorf("  Lua strings are byte sequences; reverse must be byte-level")
	}
}

// ============================================================
// BUG 2: setmetatable doesn't check __metatable protection
// When a metatable has __metatable field, setmetatable should error.
// ============================================================
func TestBug_SetmetatableProtection(t *testing.T) {
	_, _, err := runLua(t, `
		local t = setmetatable({}, {__metatable = "protected"})
		-- This should raise "cannot change a protected metatable"
		setmetatable(t, {})
	`)

	if err == nil {
		t.Error("setmetatable should reject changing a protected metatable (__metatable set)")
		t.Error("  In Lua 5.4, setmetatable raises 'cannot change a protected metatable'")
		t.Error("  but golua silently allows it")
	}
}

// ============================================================
// BUG 3: tonumber("0xff") returns nil instead of 255
// Lua 5.4: tonumber("0xff") should parse hex strings at base 10
// ============================================================
func TestBug_TonumberHex(t *testing.T) {
	results := mustRun(t, `return tonumber("0xff")`)

	if len(results) == 0 || results[0].IsNil() {
		t.Error("tonumber('0xff') returned nil, expected 255")
		t.Error("  Lua 5.4 spec: tonumber with base 10 should accept hex '0x' prefix")
	} else {
		val, ok := results[0].ToInt()
		if !ok || val != 255 {
			t.Errorf("tonumber('0xff') returned %v, expected 255", results[0])
		}
	}

	// Also test uppercase
	results2 := mustRun(t, `return tonumber("0XFF")`)
	if len(results2) == 0 || results2[0].IsNil() {
		t.Error("tonumber('0XFF') returned nil, expected 255")
	}

	// Test with spaces
	results3 := mustRun(t, `return tonumber("  0xff  ")`)
	if len(results3) == 0 || results3[0].IsNil() {
		t.Error("tonumber('  0xff  ') returned nil, expected 255")
	}
}

// ============================================================
// BUG 4: assert() doesn't return all its arguments
// Lua 5.4: assert(v, msg) returns v, msg on success
// ============================================================
func TestBug_AssertReturnsAllArgs(t *testing.T) {
	results := mustRun(t, `return assert(1, 2, 3)`)

	if len(results) != 3 {
		t.Errorf("assert(1, 2, 3) returned %d values, expected 3", len(results))
		for i, r := range results {
			t.Logf("  result[%d] = %v", i, r)
		}
		return
	}

	if results[0].AsInt() != 1 || results[1].AsInt() != 2 || results[2].AsInt() != 3 {
		t.Errorf("assert(1, 2, 3) = (%v, %v, %v), expected (1, 2, 3)",
			results[0], results[1], results[2])
	}
}

// ============================================================
// BUG 5: Closures in for loops capture same variable
// Each iteration should capture its own copy of the loop variable
// ============================================================
func TestBug_ClosureForLoopCapture(t *testing.T) {
	results := mustRun(t, `
		local funcs = {}
		for i = 1, 5 do
			funcs[i] = function() return i end
		end
		return funcs[1](), funcs[3](), funcs[5]()
	`)

	if len(results) < 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	v1 := results[0].AsInt()
	v3 := results[1].AsInt()
	v5 := results[2].AsInt()

	if v1 != 1 || v3 != 3 || v5 != 5 {
		t.Errorf("Closures in for loop all captured same variable")
		t.Errorf("  funcs[1]() = %d (expected 1)", v1)
		t.Errorf("  funcs[3]() = %d (expected 3)", v3)
		t.Errorf("  funcs[5]() = %d (expected 5)", v5)
		if v1 == v3 && v3 == v5 {
			t.Error("  All closures return the same value — loop variable not captured per-iteration")
		}
	}
}

// ============================================================
// BUG 6: pcall(non_function) crashes instead of returning error
// ============================================================
func TestBug_PcallNonFunction(t *testing.T) {
	results, _, err := runLua(t, `return pcall(42)`)

	if err != nil {
		t.Errorf("pcall(42) crashed the VM instead of returning (false, error)")
		t.Errorf("  Error: %v", err)
		return
	}

	if len(results) < 1 {
		t.Fatal("pcall(42) returned no results")
	}

	if results[0].ToBool() {
		t.Error("pcall(42) returned true, expected false")
	}
}

// ============================================================
// BUG 7: coroutine.wrap — calling after dead should error gracefully
// ============================================================
func TestBug_CoroutineWrapDead(t *testing.T) {
	results, _, err := runLua(t, `
		local gen = coroutine.wrap(function()
			coroutine.yield(1)
			return 2
		end)
		local v1 = gen()  -- yields 1
		local v2 = gen()  -- returns 2, coroutine dies
		-- Calling gen() again should error, not crash
		local ok, err = pcall(gen)
		return v1, v2, ok, tostring(err)
	`)

	if err != nil {
		t.Errorf("Calling dead coroutine.wrap crashed: %v", err)
		return
	}

	if len(results) < 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	if results[2].ToBool() {
		t.Error("Calling dead coroutine.wrap should fail, but pcall returned true")
	}
}

// ============================================================
// Additional edge case tests (not bugs, but coverage)
// ============================================================

func TestEdge_StringSubNegative(t *testing.T) {
	results := mustRun(t, `return string.sub("hello", -3)`)
	if len(results) == 0 || results[0].AsString() != "llo" {
		t.Errorf("string.sub('hello', -3) = %v, expected 'llo'", results[0])
	}
}

func TestEdge_StringRepWithSep(t *testing.T) {
	results := mustRun(t, `return string.rep("ab", 3, "-")`)
	if len(results) == 0 || results[0].AsString() != "ab-ab-ab" {
		t.Errorf("string.rep('ab', 3, '-') = %v, expected 'ab-ab-ab'", results[0])
	}
}

func TestEdge_SelectNegative(t *testing.T) {
	results := mustRun(t, `return select(-1, "a", "b", "c")`)
	if len(results) == 0 || results[0].AsString() != "c" {
		t.Errorf("select(-1, 'a', 'b', 'c') = %v, expected 'c'", results[0])
	}
}

func TestEdge_IpairsStopsAtNil(t *testing.T) {
	results := mustRun(t, `
		local t = {10, 20, nil, 40, 50}
		local count = 0
		for i, v in ipairs(t) do count = count + 1 end
		return count
	`)
	if len(results) == 0 || results[0].AsInt() != 2 {
		t.Errorf("ipairs should stop at first nil: got count=%v, expected 2", results[0])
	}
}

func TestEdge_BooleanZeroTruthy(t *testing.T) {
	// In Lua, 0 is truthy (unlike C)
	results := mustRun(t, `
		if 0 then return "truthy" else return "falsy" end
	`)
	if len(results) == 0 || results[0].AsString() != "truthy" {
		t.Error("0 should be truthy in Lua")
	}
}

func TestEdge_EmptyStringTruthy(t *testing.T) {
	results := mustRun(t, `
		if "" then return "truthy" else return "falsy" end
	`)
	if len(results) == 0 || results[0].AsString() != "truthy" {
		t.Error("Empty string should be truthy in Lua")
	}
}

func TestEdge_StringNumberEquality(t *testing.T) {
	// In Lua 5.4, "42" ~= 42 (different types)
	results := mustRun(t, `return "42" == 42`)
	if len(results) == 0 || results[0].ToBool() {
		t.Error("'42' == 42 should be false in Lua 5.4 (different types)")
	}
}

func TestEdge_CompareStringNumber(t *testing.T) {
	// In Lua 5.4, comparing string < number should error
	_, _, err := runLua(t, `return "10" < 5`)
	if err == nil {
		t.Error("Comparing string < number should raise an error in Lua 5.4")
	}
}

func TestEdge_MultipleReturnInTable(t *testing.T) {
	results := mustRun(t, `
		local function multi() return 1, 2, 3 end
		-- Last element gets all returns
		local t = {multi()}
		return #t, t[1], t[2], t[3]
	`)
	if len(results) < 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}
	if results[0].AsInt() != 3 || results[1].AsInt() != 1 || results[2].AsInt() != 2 || results[3].AsInt() != 3 {
		t.Errorf("Table from multi-return: got %v,%v,%v,%v expected 3,1,2,3",
			results[0], results[1], results[2], results[3])
	}
}

func TestEdge_MultipleReturnTruncated(t *testing.T) {
	results := mustRun(t, `
		local function multi() return 1, 2, 3 end
		-- Not last: only first value used
		local t = {multi(), "x"}
		return #t, t[1], t[2]
	`)
	if len(results) < 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	if results[0].AsInt() != 2 || results[1].AsInt() != 1 || results[2].AsString() != "x" {
		t.Errorf("Multi-return truncation: got len=%v,t[1]=%v,t[2]=%v expected 2,1,'x'",
			results[0], results[1], results[2])
	}
}

func TestEdge_NestedPcall(t *testing.T) {
	results := mustRun(t, `
		local ok, err = pcall(function()
			local ok2, err2 = pcall(function() error("inner") end)
			assert(ok2 == false)
			error("outer")
		end)
		return ok, tostring(err)
	`)
	if len(results) < 2 {
		t.Fatal("Expected 2 results")
	}
	if results[0].ToBool() {
		t.Error("Outer pcall should return false")
	}
	if !strings.Contains(results[1].AsString(), "outer") {
		t.Errorf("Expected 'outer' in error, got: %s", results[1].AsString())
	}
}

func TestEdge_ErrorWithObject(t *testing.T) {
	results := mustRun(t, `
		local ok, err = pcall(function()
			error({code = 404, msg = "not found"})
		end)
		return ok, type(err), err.code
	`)
	if len(results) < 3 {
		t.Fatal("Expected 3 results")
	}
	if results[0].ToBool() {
		t.Error("pcall should return false")
	}
	if results[1].AsString() != "table" {
		t.Errorf("Error object type: got %s, expected table", results[1].AsString())
	}
	if results[2].AsInt() != 404 {
		t.Errorf("Error code: got %v, expected 404", results[2])
	}
}

func TestEdge_MetatableChain(t *testing.T) {
	results := mustRun(t, `
		local base = {x = 1}
		local derived = setmetatable({y = 2}, {__index = base})
		local leaf = setmetatable({z = 3}, {__index = derived})
		return leaf.x, leaf.y, leaf.z, leaf.w == nil
	`)
	if len(results) < 4 {
		t.Fatal("Expected 4 results")
	}
	if results[0].AsInt() != 1 {
		t.Errorf("leaf.x = %v, expected 1", results[0])
	}
	if results[1].AsInt() != 2 {
		t.Errorf("leaf.y = %v, expected 2", results[1])
	}
	if results[2].AsInt() != 3 {
		t.Errorf("leaf.z = %v, expected 3", results[2])
	}
	if !results[3].ToBool() {
		t.Error("leaf.w should be nil")
	}
}

func TestEdge_GsubTableReplacement(t *testing.T) {
	results := mustRun(t, `
		return string.gsub("hello world", "%w+", {hello = "HI", world = "THERE"})
	`)
	if len(results) == 0 || results[0].AsString() != "HI THERE" {
		t.Errorf("gsub with table: got %v, expected 'HI THERE'", results[0])
	}
}

func TestEdge_GsubCountLimit(t *testing.T) {
	results := mustRun(t, `
		local r, n = string.gsub("aaa", "a", "b", 2)
		return r, n
	`)
	if len(results) < 2 {
		t.Fatal("Expected 2 results")
	}
	if results[0].AsString() != "bba" {
		t.Errorf("gsub with limit: got %q, expected 'bba'", results[0].AsString())
	}
	if results[1].AsInt() != 2 {
		t.Errorf("gsub count: got %v, expected 2", results[1])
	}
}

func TestEdge_ScalarTimesMetamethod(t *testing.T) {
	// When number * table, the table's __mul metamethod should be called
	results := mustRun(t, `
		local mt = {
			__mul = function(a, b)
				if type(a) == "number" then return a * b.val
				else return a.val * b end
			end
		}
		local obj = setmetatable({val = 10}, mt)
		return 3 * obj, obj * 5
	`)
	if len(results) < 2 {
		t.Fatal("Expected 2 results")
	}
	if results[0].AsInt() != 30 {
		t.Errorf("3 * obj = %v, expected 30", results[0])
	}
	if results[1].AsInt() != 50 {
		t.Errorf("obj * 5 = %v, expected 50", results[1])
	}
}

func TestEdge_BalancedPattern(t *testing.T) {
	results := mustRun(t, `
		return string.match("(hello (world))", "%b()")
	`)
	if len(results) == 0 || results[0].AsString() != "(hello (world))" {
		t.Errorf("Balanced pattern: got %v, expected '(hello (world))'", results[0])
	}
}

func TestEdge_StringFindPlainSpecialChars(t *testing.T) {
	results := mustRun(t, `
		local s, e = string.find("a.b.c", ".", 1, true)
		return s, e
	`)
	if len(results) < 2 || results[0].AsInt() != 2 {
		t.Errorf("string.find plain '.': got start=%v, expected 2", results[0])
	}
}

func TestEdge_FormatPercent(t *testing.T) {
	results := mustRun(t, `return string.format("100%%")`)
	if len(results) == 0 || results[0].AsString() != "100%" {
		t.Errorf("format %%%%: got %q, expected '100%%'", results[0].AsString())
	}
}

func TestEdge_LoadWithEnv(t *testing.T) {
	results := mustRun(t, `
		local env = {x = 100}
		setmetatable(env, {__index = _G})
		local fn = load("return x", "test", "t", env)
		return fn()
	`)
	if len(results) == 0 || results[0].AsInt() != 100 {
		t.Errorf("load with custom env: got %v, expected 100", results[0])
	}
}

func TestEdge_DeepNestedClosures(t *testing.T) {
	results := mustRun(t, `
		local function outer(x)
			local function middle(y)
				local function inner(z)
					return x + y + z
				end
				return inner
			end
			return middle
		end
		return outer(1)(2)(3)
	`)
	if len(results) == 0 || results[0].AsInt() != 6 {
		t.Errorf("Nested closures: got %v, expected 6", results[0])
	}
}

func TestEdge_CoroutineStatus(t *testing.T) {
	results := mustRun(t, `
		local co = coroutine.create(function(a, b)
			coroutine.yield(a + b)
			return a * b
		end)
		local s1 = coroutine.status(co)
		local ok1, v1 = coroutine.resume(co, 10, 3)
		local s2 = coroutine.status(co)
		local ok2, v2 = coroutine.resume(co)
		local s3 = coroutine.status(co)
		local ok3, _ = coroutine.resume(co)
		return s1, ok1, v1, s2, ok2, v2, s3, ok3
	`)
	if len(results) < 8 {
		t.Fatalf("Expected 8 results, got %d", len(results))
	}
	if results[0].AsString() != "suspended" {
		t.Errorf("Initial status: %s, expected suspended", results[0].AsString())
	}
	if !results[1].ToBool() || results[2].AsInt() != 13 {
		t.Error("First resume failed")
	}
	if results[3].AsString() != "suspended" {
		t.Errorf("After yield status: %s, expected suspended", results[3].AsString())
	}
	if !results[4].ToBool() || results[5].AsInt() != 30 {
		t.Error("Second resume failed")
	}
	if results[6].AsString() != "dead" {
		t.Errorf("Final status: %s, expected dead", results[6].AsString())
	}
	if results[7].ToBool() {
		t.Error("Resume dead coroutine should return false")
	}
}

func TestEdge_RepeatUntilScope(t *testing.T) {
	// The repeat-until condition can see locals declared in the body
	results := mustRun(t, `
		local n = 3
		repeat
			local x = n * 2
			n = n - 1
		until x == 2
		return n
	`)
	if len(results) == 0 || results[0].AsInt() != 0 {
		t.Errorf("repeat-until scope: got n=%v, expected 0", results[0])
	}
}

func TestEdge_TonumberBase(t *testing.T) {
	results := mustRun(t, `
		return tonumber("ff", 16), tonumber("1010", 2), tonumber("77", 8)
	`)
	if len(results) < 3 {
		t.Fatal("Expected 3 results")
	}
	if results[0].AsInt() != 255 {
		t.Errorf("tonumber('ff', 16) = %v, expected 255", results[0])
	}
	if results[1].AsInt() != 10 {
		t.Errorf("tonumber('1010', 2) = %v, expected 10", results[1])
	}
	if results[2].AsInt() != 63 {
		t.Errorf("tonumber('77', 8) = %v, expected 63", results[2])
	}
}

// ============================================================
// Adjacent tests for bug fixes
// ============================================================

func TestAdj_StringUpperLowerNonASCII(t *testing.T) {
	// string.upper/lower must only affect ASCII a-z/A-Z, leave other bytes intact
	results := mustRun(t, `
		local s = "heLLo\xC3\xA9"  -- mixed case + non-ASCII bytes
		return string.upper(s), string.lower(s)
	`)
	if len(results) < 2 {
		t.Fatal("Expected 2 results")
	}
	upper := results[0].AsString()
	lower := results[1].AsString()
	// upper should uppercase ASCII only, non-ASCII bytes unchanged
	if upper != "HELLO\xC3\xA9" {
		t.Errorf("string.upper non-ASCII: got %q, expected %q", upper, "HELLO\xC3\xA9")
	}
	if lower != "hello\xC3\xA9" {
		t.Errorf("string.lower non-ASCII: got %q, expected %q", lower, "hello\xC3\xA9")
	}
}

func TestAdj_StringReverseASCII(t *testing.T) {
	results := mustRun(t, `return string.reverse("abcde")`)
	if len(results) == 0 || results[0].AsString() != "edcba" {
		t.Errorf("string.reverse('abcde') = %v, expected 'edcba'", results[0])
	}
}

func TestAdj_GetmetatableProtected(t *testing.T) {
	// getmetatable should return __metatable value, not the real metatable
	results := mustRun(t, `
		local mt = {__metatable = "protected"}
		local t = setmetatable({}, mt)
		return getmetatable(t)
	`)
	if len(results) == 0 || results[0].AsString() != "protected" {
		t.Errorf("getmetatable with __metatable: got %v, expected 'protected'", results[0])
	}
}

func TestAdj_SetmetatableProtectedError(t *testing.T) {
	// Verify the error message matches Lua 5.4
	_, _, err := runLua(t, `
		local t = setmetatable({}, {__metatable = "no touch"})
		setmetatable(t, {})
	`)
	if err == nil {
		t.Fatal("Expected error from setmetatable on protected table")
	}
	if !strings.Contains(err.Error(), "cannot change a protected metatable") {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestAdj_TonumberNegativeHex(t *testing.T) {
	// tonumber("-0xff") — Lua 5.4 does not support negative hex prefix
	// but tonumber with explicit base should work for hex
	results := mustRun(t, `return tonumber("0xA0")`)
	if len(results) == 0 || results[0].IsNil() {
		t.Error("tonumber('0xA0') returned nil, expected 160")
	} else if results[0].AsInt() != 160 {
		t.Errorf("tonumber('0xA0') = %v, expected 160", results[0])
	}
}

func TestAdj_AssertSingleArg(t *testing.T) {
	results := mustRun(t, `return assert(42)`)
	if len(results) != 1 {
		t.Errorf("assert(42) returned %d values, expected 1", len(results))
	}
	if results[0].AsInt() != 42 {
		t.Errorf("assert(42) = %v, expected 42", results[0])
	}
}

func TestAdj_AssertFalseMessage(t *testing.T) {
	_, _, err := runLua(t, `assert(false, "custom error")`)
	if err == nil {
		t.Fatal("assert(false) should error")
	}
	if !strings.Contains(err.Error(), "custom error") {
		t.Errorf("assert error: got %v, expected 'custom error'", err)
	}
}

func TestAdj_ForLoopClosureNested(t *testing.T) {
	// Nested for loops — each level should capture its own variable
	results := mustRun(t, `
		local funcs = {}
		for i = 1, 3 do
			for j = 1, 3 do
				funcs[#funcs + 1] = function() return i, j end
			end
		end
		local a, b = funcs[1]()   -- should be 1, 1
		local c, d = funcs[5]()   -- should be 2, 2
		local e, f = funcs[9]()   -- should be 3, 3
		return a, b, c, d, e, f
	`)
	if len(results) < 6 {
		t.Fatalf("Expected 6 results, got %d", len(results))
	}
	if results[0].AsInt() != 1 || results[1].AsInt() != 1 {
		t.Errorf("funcs[1]() = (%v,%v), expected (1,1)", results[0], results[1])
	}
	if results[2].AsInt() != 2 || results[3].AsInt() != 2 {
		t.Errorf("funcs[5]() = (%v,%v), expected (2,2)", results[2], results[3])
	}
	if results[4].AsInt() != 3 || results[5].AsInt() != 3 {
		t.Errorf("funcs[9]() = (%v,%v), expected (3,3)", results[4], results[5])
	}
}

func TestAdj_GenericForClosureCapture(t *testing.T) {
	// Generic for (ipairs) should also capture per-iteration
	results := mustRun(t, `
		local funcs = {}
		for i, v in ipairs({10, 20, 30}) do
			funcs[i] = function() return v end
		end
		return funcs[1](), funcs[2](), funcs[3]()
	`)
	if len(results) < 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	if results[0].AsInt() != 10 {
		t.Errorf("funcs[1]() = %v, expected 10", results[0])
	}
	if results[1].AsInt() != 20 {
		t.Errorf("funcs[2]() = %v, expected 20", results[1])
	}
	if results[2].AsInt() != 30 {
		t.Errorf("funcs[3]() = %v, expected 30", results[2])
	}
}
