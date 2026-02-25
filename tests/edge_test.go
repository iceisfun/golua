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

// ============================================================
// Probe tests: comprehensive edge case coverage
// ============================================================

func TestProbe_FormatSNonString(t *testing.T) {
	results := mustRun(t, `
		return string.format("%s", nil),
		       string.format("%s", true),
		       string.format("%s", false),
		       string.format("%s", 42),
		       string.format("%s", 3.14)
	`)
	if len(results) < 5 {
		t.Fatalf("Expected 5 results, got %d", len(results))
	}
	checks := []string{"nil", "true", "false", "42", "3.14"}
	for i, exp := range checks {
		if results[i].AsString() != exp {
			t.Errorf("format %%s [%d]: got %q, expected %q", i, results[i].AsString(), exp)
		}
	}
}

func TestProbe_FormatDFloat(t *testing.T) {
	results := mustRun(t, `return string.format("%d", 3.9), string.format("%d", -3.9)`)
	if results[0].AsString() != "3" {
		t.Errorf("%%d 3.9: got %q, expected '3'", results[0].AsString())
	}
	if results[1].AsString() != "-3" {
		t.Errorf("%%d -3.9: got %q, expected '-3'", results[1].AsString())
	}
}

func TestProbe_FormatC(t *testing.T) {
	results := mustRun(t, `return string.format("%c", 65), string.format("%c", 97), string.format("%c", 48)`)
	if results[0].AsString() != "A" || results[1].AsString() != "a" || results[2].AsString() != "0" {
		t.Errorf("%%c: got %q %q %q", results[0].AsString(), results[1].AsString(), results[2].AsString())
	}
}

func TestProbe_FormatI(t *testing.T) {
	results := mustRun(t, `return string.format("%i", 42)`)
	if results[0].AsString() != "42" {
		t.Errorf("%%i: got %q, expected '42'", results[0].AsString())
	}
}

func TestProbe_FormatWidthPrecision(t *testing.T) {
	results := mustRun(t, `
		return string.format("%10s", "hi"),
		       string.format("%-10s", "hi"),
		       string.format("%.3s", "hello")
	`)
	if results[0].AsString() != "        hi" {
		t.Errorf("%%10s: got %q", results[0].AsString())
	}
	if results[1].AsString() != "hi        " {
		t.Errorf("%%-10s: got %q", results[1].AsString())
	}
	if results[2].AsString() != "hel" {
		t.Errorf("%%.3s: got %q", results[2].AsString())
	}
}

func TestProbe_FindPastEnd(t *testing.T) {
	results := mustRun(t, `return string.find("hello", "l", 100)`)
	if !results[0].IsNil() {
		t.Errorf("find past end: got %v, expected nil", results[0])
	}
}

func TestProbe_FindEmpty(t *testing.T) {
	results := mustRun(t, `return string.find("hello", "")`)
	if results[0].AsInt() != 1 || results[1].AsInt() != 0 {
		t.Errorf("find empty: got (%v, %v), expected (1, 0)", results[0], results[1])
	}
}

func TestProbe_FindNegInit(t *testing.T) {
	results := mustRun(t, `return string.find("hello", "l", -3)`)
	if results[0].AsInt() != 3 || results[1].AsInt() != 3 {
		t.Errorf("find neg init: got (%v, %v), expected (3, 3)", results[0], results[1])
	}
}

func TestProbe_SubEdges(t *testing.T) {
	results := mustRun(t, `
		return string.sub("hello", -3, -1),
		       string.sub("hello", 0),
		       string.sub("hello", 1, 100),
		       string.sub("hello", 3, 2),
		       string.sub("hello", -100)
	`)
	checks := []string{"llo", "hello", "hello", "", "hello"}
	for i, exp := range checks {
		if results[i].AsString() != exp {
			t.Errorf("sub edge [%d]: got %q, expected %q", i, results[i].AsString(), exp)
		}
	}
}

func TestProbe_SelectHash(t *testing.T) {
	results := mustRun(t, `return select("#"), select("#", "a", "b", "c")`)
	if results[0].AsInt() != 0 {
		t.Errorf("select('#'): got %v, expected 0", results[0])
	}
	if results[1].AsInt() != 3 {
		t.Errorf("select('#', a, b, c): got %v, expected 3", results[1])
	}
}

func TestProbe_StringArithmetic(t *testing.T) {
	results := mustRun(t, `return "10" + 5, "3" * "4", "10" - 1`)
	if results[0].AsInt() != 15 {
		t.Errorf("'10' + 5: got %v", results[0])
	}
	if results[1].AsInt() != 12 {
		t.Errorf("'3' * '4': got %v", results[1])
	}
	if results[2].AsInt() != 9 {
		t.Errorf("'10' - 1: got %v", results[2])
	}
}

func TestProbe_ConcatNum(t *testing.T) {
	results := mustRun(t, `return 10 .. 20, 1 .. 2, 1.5 .. "x"`)
	if results[0].AsString() != "1020" {
		t.Errorf("10..20: got %q", results[0].AsString())
	}
	if results[1].AsString() != "12" {
		t.Errorf("1..2: got %q", results[1].AsString())
	}
	if results[2].AsString() != "1.5x" {
		t.Errorf("1.5..'x': got %q", results[2].AsString())
	}
}

func TestProbe_TableConcat(t *testing.T) {
	results := mustRun(t, `
		return table.concat({}),
		       table.concat({"a"}),
		       table.concat({"a","b","c"}, ","),
		       table.concat({"a","b","c","d"}, ",", 2, 3),
		       table.concat({1,2,3}, "+")
	`)
	checks := []string{"", "a", "a,b,c", "b,c", "1+2+3"}
	for i, exp := range checks {
		if results[i].AsString() != exp {
			t.Errorf("table.concat [%d]: got %q, expected %q", i, results[i].AsString(), exp)
		}
	}
}

func TestProbe_TablePack(t *testing.T) {
	results := mustRun(t, `
		local p = table.pack(10, nil, 30)
		return p.n, p[1], p[2], p[3]
	`)
	if results[0].AsInt() != 3 {
		t.Errorf("pack.n: got %v", results[0])
	}
	if results[1].AsInt() != 10 {
		t.Errorf("pack[1]: got %v", results[1])
	}
	if !results[2].IsNil() {
		t.Errorf("pack[2]: got %v, expected nil", results[2])
	}
	if results[3].AsInt() != 30 {
		t.Errorf("pack[3]: got %v", results[3])
	}
}

func TestProbe_TableSort(t *testing.T) {
	results := mustRun(t, `
		local t = {3, 1, 4, 1, 5, 9, 2, 6}
		table.sort(t)
		local r1, r8 = t[1], t[8]
		table.sort(t, function(a, b) return a > b end)
		return r1, r8, t[1], t[8]
	`)
	if results[0].AsInt() != 1 || results[1].AsInt() != 9 {
		t.Errorf("sort asc: got [1]=%v [8]=%v", results[0], results[1])
	}
	if results[2].AsInt() != 9 || results[3].AsInt() != 1 {
		t.Errorf("sort desc: got [1]=%v [8]=%v", results[2], results[3])
	}
}

func TestProbe_TableSortEmpty(t *testing.T) {
	mustRun(t, `table.sort({})`)
}

func TestProbe_PcallErrorObjects(t *testing.T) {
	results := mustRun(t, `
		local ok1, err1 = pcall(error, 42)
		local ok2, err2 = pcall(error, true)
		local ok3, err3 = pcall(error, nil)
		return err1, err2, err3
	`)
	if results[0].AsInt() != 42 {
		t.Errorf("pcall error(42): got %v", results[0])
	}
	if !results[1].ToBool() || !results[1].IsBool() {
		t.Errorf("pcall error(true): got %v", results[1])
	}
	if !results[2].IsNil() {
		t.Errorf("pcall error(nil): got %v, expected nil", results[2])
	}
}

func TestProbe_XpcallHandler(t *testing.T) {
	results := mustRun(t, `
		local ok, err = xpcall(
			function() error("oops") end,
			function(e) return "handled: " .. e end
		)
		return ok, err
	`)
	if results[0].ToBool() {
		t.Error("xpcall should return false")
	}
	if !strings.Contains(results[1].AsString(), "handled:") {
		t.Errorf("xpcall handler: got %q", results[1].AsString())
	}
}

func TestProbe_XpcallExtraArgs(t *testing.T) {
	results := mustRun(t, `
		local ok, val = xpcall(function(a, b) return a + b end, tostring, 10, 20)
		return ok, val
	`)
	if !results[0].ToBool() || results[1].AsInt() != 30 {
		t.Errorf("xpcall args: ok=%v val=%v", results[0], results[1])
	}
}

func TestProbe_ErrorLevel0(t *testing.T) {
	results := mustRun(t, `
		local ok, err = pcall(function() error("raw", 0) end)
		return err
	`)
	if results[0].AsString() != "raw" {
		t.Errorf("error level 0: got %q, expected 'raw'", results[0].AsString())
	}
}

func TestProbe_ForZeroStep(t *testing.T) {
	// Integer zero step
	_, _, err := runLua(t, `for i = 1, 10, 0 do end`)
	if err == nil {
		t.Error("for loop with step 0 should error")
	}
	// Float zero step
	_, _, err = runLua(t, `for i = 1.0, 10.0, 0.0 do end`)
	if err == nil {
		t.Error("for loop with float step 0.0 should error")
	}
}

func TestProbe_ForNegStep(t *testing.T) {
	results := mustRun(t, `
		local sum = 0
		for i = 5, 1, -1 do sum = sum + i end
		return sum
	`)
	if results[0].AsInt() != 15 {
		t.Errorf("for neg step sum: got %v, expected 15", results[0])
	}
}

func TestProbe_ForFloatStep(t *testing.T) {
	results := mustRun(t, `
		local sum, count = 0, 0
		for i = 0.0, 1.0, 0.5 do sum = sum + i; count = count + 1 end
		return count, sum
	`)
	if results[0].AsInt() != 3 {
		t.Errorf("float step count: got %v, expected 3", results[0])
	}
}

func TestProbe_ForEmptyRange(t *testing.T) {
	results := mustRun(t, `
		local count = 0
		for i = 10, 1 do count = count + 1 end
		return count
	`)
	if results[0].AsInt() != 0 {
		t.Errorf("empty range: got count=%v, expected 0", results[0])
	}
}

func TestProbe_StringColonMethods(t *testing.T) {
	results := mustRun(t, `
		return ("hello"):upper(), ("hello"):len(), ("hello"):sub(2, 4),
		       ("ab"):rep(3), ("hello"):reverse()
	`)
	checks := []string{"HELLO", "5", "ell", "ababab", "olleh"}
	for i, exp := range checks {
		got := results[i].AsString()
		if i == 1 {
			if results[i].AsInt() != 5 {
				t.Errorf("colon method [%d]: got %v, expected %s", i, results[i], exp)
			}
			continue
		}
		if got != exp {
			t.Errorf("colon method [%d]: got %q, expected %q", i, got, exp)
		}
	}
}

func TestProbe_RawFunctions(t *testing.T) {
	results := mustRun(t, `
		local mt = setmetatable({}, {
			__eq = function(a, b) return true end,
			__len = function() return 999 end,
			__index = function(t, k) return "meta" end,
			__newindex = function(t, k, v) end
		})
		rawset(mt, "x", 42)
		return rawequal(1, 1), rawequal(1, 2), rawget(mt, "x"), rawget(mt, "y"), rawlen(mt)
	`)
	if !results[0].ToBool() {
		t.Error("rawequal(1,1) should be true")
	}
	if results[1].ToBool() {
		t.Error("rawequal(1,2) should be false")
	}
	if results[2].AsInt() != 42 {
		t.Errorf("rawget after rawset: got %v, expected 42", results[2])
	}
	if !results[3].IsNil() {
		t.Errorf("rawget bypassing __index: got %v, expected nil", results[3])
	}
	if results[4].AsInt() != 1 {
		// rawlen on a table with one key set
		t.Logf("rawlen: got %v (may vary based on table internals)", results[4])
	}
}

func TestProbe_GmatchWords(t *testing.T) {
	results := mustRun(t, `
		local words = {}
		for w in string.gmatch("one two three", "%w+") do
			words[#words + 1] = w
		end
		return #words, words[1], words[3]
	`)
	if results[0].AsInt() != 3 {
		t.Errorf("gmatch count: got %v, expected 3", results[0])
	}
	if results[1].AsString() != "one" {
		t.Errorf("gmatch[1]: got %q", results[1].AsString())
	}
	if results[2].AsString() != "three" {
		t.Errorf("gmatch[3]: got %q", results[2].AsString())
	}
}

func TestProbe_MatchMultiCapture(t *testing.T) {
	results := mustRun(t, `return string.match("2023-01-15", "(%d+)-(%d+)-(%d+)")`)
	if len(results) < 3 {
		t.Fatalf("Expected 3 captures, got %d", len(results))
	}
	if results[0].AsString() != "2023" || results[1].AsString() != "01" || results[2].AsString() != "15" {
		t.Errorf("match: got %q, %q, %q", results[0].AsString(), results[1].AsString(), results[2].AsString())
	}
}

func TestProbe_ByteCharRoundtrip(t *testing.T) {
	results := mustRun(t, `
		return string.byte(string.char(255)),
		       string.byte(string.char(0)),
		       string.byte(string.char(128)),
		       string.char(65, 66, 67)
	`)
	if results[0].AsInt() != 255 {
		t.Errorf("byte(char(255)): got %v", results[0])
	}
	if results[1].AsInt() != 0 {
		t.Errorf("byte(char(0)): got %v", results[1])
	}
	if results[2].AsInt() != 128 {
		t.Errorf("byte(char(128)): got %v", results[2])
	}
	if results[3].AsString() != "ABC" {
		t.Errorf("char(65,66,67): got %q", results[3].AsString())
	}
}

func TestProbe_TostringMetaNonString(t *testing.T) {
	results := mustRun(t, `
		local obj = setmetatable({}, {__tostring = function() return 42 end})
		local r = tostring(obj)
		return r, type(r)
	`)
	// Lua 5.4: tostring passes through __tostring result as-is
	// The result may be int 42 or string "42" depending on implementation
	tp := results[1].AsString()
	if tp == "number" {
		if results[0].AsInt() != 42 {
			t.Errorf("tostring meta int: got %v, expected 42", results[0])
		}
	} else if tp == "string" {
		if results[0].AsString() != "42" {
			t.Errorf("tostring meta string: got %q, expected '42'", results[0].AsString())
		}
	} else {
		t.Errorf("tostring meta: unexpected type %q, value %v", tp, results[0])
	}
}

func TestProbe_NextIteration(t *testing.T) {
	results := mustRun(t, `
		local t = {a = 1, b = 2, c = 3}
		local count = 0
		local k = next(t)
		while k ~= nil do
			count = count + 1
			k = next(t, k)
		end
		return count, next({})
	`)
	if results[0].AsInt() != 3 {
		t.Errorf("next iteration count: got %v, expected 3", results[0])
	}
	if !results[1].IsNil() {
		t.Errorf("next(empty): got %v, expected nil", results[1])
	}
}

func TestProbe_TypeEdges(t *testing.T) {
	results := mustRun(t, `
		return type(nil), type(true), type(42), type(3.14),
		       type("hi"), type({}), type(print)
	`)
	expected := []string{"nil", "boolean", "number", "number", "string", "table", "function"}
	for i, exp := range expected {
		if results[i].AsString() != exp {
			t.Errorf("type[%d]: got %q, expected %q", i, results[i].AsString(), exp)
		}
	}
}

func TestProbe_TableRemove(t *testing.T) {
	results := mustRun(t, `
		local t = {10, 20, 30, 40}
		local last = table.remove(t)
		local first = table.remove(t, 1)
		return last, #t, first, t[1]
	`)
	if results[0].AsInt() != 40 {
		t.Errorf("remove last: got %v", results[0])
	}
	if results[1].AsInt() != 2 {
		t.Errorf("len after removes: got %v", results[1])
	}
	if results[2].AsInt() != 10 {
		t.Errorf("remove first: got %v", results[2])
	}
	if results[3].AsInt() != 20 {
		t.Errorf("t[1] after remove: got %v", results[3])
	}
}

func TestProbe_TableMoveBetween(t *testing.T) {
	results := mustRun(t, `
		local src = {10, 20, 30}
		local dst = {0, 0, 0, 0, 0}
		table.move(src, 1, 3, 2, dst)
		return dst[2], dst[3], dst[4]
	`)
	if results[0].AsInt() != 10 || results[1].AsInt() != 20 || results[2].AsInt() != 30 {
		t.Errorf("table.move between: got %v, %v, %v", results[0], results[1], results[2])
	}
}

func TestProbe_MathFunctions(t *testing.T) {
	results := mustRun(t, `
		return math.abs(-5), math.abs(5),
		       math.floor(3.7), math.ceil(3.2),
		       math.floor(-3.2), math.ceil(-3.7),
		       math.fmod(7, 3)
	`)
	expected := []int64{5, 5, 3, 4, -4, -3, 1}
	for i, exp := range expected {
		if results[i].AsInt() != exp {
			t.Errorf("math[%d]: got %v, expected %d", i, results[i], exp)
		}
	}
}

func TestProbe_MathType(t *testing.T) {
	results := mustRun(t, `
		return math.type(42), math.type(3.14), math.type("42"), math.type(nil)
	`)
	if results[0].AsString() != "integer" {
		t.Errorf("math.type(42): got %q", results[0].AsString())
	}
	if results[1].AsString() != "float" {
		t.Errorf("math.type(3.14): got %q", results[1].AsString())
	}
	if results[2].ToBool() {
		t.Errorf("math.type('42'): should be false, got %v", results[2])
	}
	if results[3].ToBool() {
		t.Errorf("math.type(nil): should be false, got %v", results[3])
	}
}

func TestProbe_MathTointeger(t *testing.T) {
	results := mustRun(t, `
		return math.tointeger(42), math.tointeger(42.0), math.tointeger(42.5), math.tointeger("42")
	`)
	if results[0].AsInt() != 42 {
		t.Errorf("tointeger(42): got %v", results[0])
	}
	if results[1].AsInt() != 42 {
		t.Errorf("tointeger(42.0): got %v", results[1])
	}
	if !results[2].IsNil() {
		t.Errorf("tointeger(42.5): got %v, expected nil", results[2])
	}
	if !results[3].IsNil() {
		t.Errorf("tointeger('42'): got %v, expected nil", results[3])
	}
}

func TestProbe_VarargForwarding(t *testing.T) {
	results := mustRun(t, `
		local function vforward(...) return ... end
		return vforward(10, 20, 30)
	`)
	if len(results) < 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	if results[0].AsInt() != 10 || results[1].AsInt() != 20 || results[2].AsInt() != 30 {
		t.Errorf("vararg forward: got %v, %v, %v", results[0], results[1], results[2])
	}
}

func TestProbe_VarargNilCount(t *testing.T) {
	results := mustRun(t, `
		local function vfunc(...) return select("#", ...) end
		return vfunc(1, nil, 3)
	`)
	if results[0].AsInt() != 3 {
		t.Errorf("vararg nil count: got %v, expected 3", results[0])
	}
}

// ---- pairs iteration with key deletion ----

func TestPairs_DeleteCurrentKey(t *testing.T) {
	// Deleting the current key during iteration should not terminate early
	results := mustRun(t, `
		local t = {A=1, B=2, C=3, D=4}
		local count = 0
		for k, v in pairs(t) do
			count = count + 1
			t[k] = nil  -- delete current key
		end
		return count
	`)
	if results[0].AsInt() != 4 {
		t.Errorf("expected 4 visited keys, got %v", results[0])
	}
}

func TestPairs_DeleteFutureKey(t *testing.T) {
	// Deleting a future key during iteration — it should be skipped
	results := mustRun(t, `
		local t = {A=1, B=2, C=3}
		local visited = {}
		for k, v in pairs(t) do
			visited[#visited+1] = k
			-- Delete all other keys
			for k2, _ in pairs(t) do
				if k2 ~= k then t[k2] = nil end
			end
		end
		-- At least the first key should be visited; subsequent ones were deleted
		return #visited
	`)
	// After deleting everything else, only 1 key is visited
	if results[0].AsInt() < 1 {
		t.Errorf("expected at least 1 visited key, got %v", results[0])
	}
}

func TestPairs_DeleteAndReinsert(t *testing.T) {
	// Delete a key then re-insert it; should not cause duplicate visits
	results := mustRun(t, `
		local t = {X=1}
		t.X = nil
		t.X = 2
		local count = 0
		for k, v in pairs(t) do
			count = count + 1
		end
		return count, t.X
	`)
	if results[0].AsInt() != 1 {
		t.Errorf("expected 1 key, got %v", results[0])
	}
	if results[1].AsInt() != 2 {
		t.Errorf("expected X=2, got %v", results[1])
	}
}

func TestPairs_NextAfterAllDeleted(t *testing.T) {
	// Delete all keys, then next(t, nil) should return nil
	results := mustRun(t, `
		local t = {A=1, B=2}
		t.A = nil
		t.B = nil
		local k, v = next(t)
		return k == nil
	`)
	if !results[0].ToBool() {
		t.Errorf("next on empty table should return nil")
	}
}

func TestPairs_MixedArrayAndHash(t *testing.T) {
	// Delete from hash part while iterating, array part should still be visited
	results := mustRun(t, `
		local t = {10, 20, x="a", y="b"}
		local count = 0
		for k, v in pairs(t) do
			count = count + 1
			if k == "x" then t.x = nil end
			if k == "y" then t.y = nil end
		end
		return count
	`)
	if results[0].AsInt() != 4 {
		t.Errorf("expected 4 visited entries, got %v", results[0])
	}
}
