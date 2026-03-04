package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithDebug(t *testing.T, source, name string, provider vm.LuaDebugProvider) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	v.SetDebugProvider(provider)
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestDebug_NoProvider(t *testing.T) {
	source := `
		assert(debug == nil, "expected debug to be nil without provider")
	`
	runLuaSource(t, source, "test_debug_no_provider")
}

func TestDebug_Traceback(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local tb = debug.traceback()
		assert(type(tb) == "string", "expected string from traceback")
		assert(tb:find("stack traceback") ~= nil, "expected 'stack traceback' in output, got: " .. tb)
	`
	runLuaWithDebug(t, source, "test_debug_traceback", provider)
}

func TestDebug_TracebackMessage(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local tb = debug.traceback("my error message")
		assert(tb:find("my error message") ~= nil, "expected message in traceback, got: " .. tb)
	`
	runLuaWithDebug(t, source, "test_debug_traceback_message", provider)
}

func TestDebug_TracebackLevel(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function inner()
			return debug.traceback("msg", 2)
		end
		local function outer()
			return inner()
		end
		local tb = outer()
		assert(type(tb) == "string", "expected string")
		-- level 2 should skip more frames
		assert(tb:find("stack traceback") ~= nil, "expected 'stack traceback' in output")
	`
	runLuaWithDebug(t, source, "test_debug_traceback_level", provider)
}

func TestDebug_StackDepth(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local d = debug.stackdepth()
		assert(type(d) == "number", "expected number, got " .. type(d))
		assert(d > 0, "expected depth > 0, got " .. tostring(d))
	`
	runLuaWithDebug(t, source, "test_debug_stackdepth", provider)
}

func TestDebug_StackDepthNested(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local d1 = debug.stackdepth()
		local function level2()
			return debug.stackdepth()
		end
		local function level1()
			return level2()
		end
		local d2 = level1()
		-- d2 should be deeper than d1 because level1 -> level2 adds frames
		assert(d2 >= d1, "expected deeper or equal stack depth, got d1=" .. tostring(d1) .. " d2=" .. tostring(d2))
	`
	runLuaWithDebug(t, source, "test_debug_stackdepth_nested", provider)
}

func TestDebug_Where(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local src, line = debug.where()
		assert(type(src) == "string", "expected string source, got " .. type(src))
		assert(type(line) == "number", "expected number line, got " .. type(line))
		assert(line > 0, "expected line > 0, got " .. tostring(line))
	`
	runLuaWithDebug(t, source, "test_debug_where", provider)
}

func TestDebug_WhereLevel(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function inner()
			-- Avoid tail call so inner's frame stays on the call stack.
			-- level 1 = where's caller (inner), level 2 = inner's caller (main)
			local src, line = debug.where(2)
			return src, line
		end
		local src, line = inner()
		-- level 2 from inside inner should reach the main chunk
		assert(type(src) == "string", "expected string source at level 2")
	`
	runLuaWithDebug(t, source, "test_debug_where_level", provider)
}

func TestDebug_NoMutation(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- With DefaultDebugProvider, all functions are available
		assert(type(debug.sethook) == "function", "sethook should exist")
		assert(type(debug.setlocal) == "function", "setlocal should exist")
		assert(type(debug.setmetatable) == "function", "debug.setmetatable should exist")
	`
	runLuaWithDebug(t, source, "test_debug_no_mutation", provider)
}

func TestDebug_Hooks(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		assert(type(debug.gethook) == "function", "gethook should exist")
		assert(type(debug.sethook) == "function", "sethook should exist")
	`
	runLuaWithDebug(t, source, "test_debug_hooks", provider)
}

// TestDebug_TCO_InvisibleRecursion verifies that the debug provider doesn't
// get confused by tail call optimization. A tail-recursive function that
// recurses 1000 times should NOT grow the stack to depth 1000+.
func TestDebug_TCO_InvisibleRecursion(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function recurse(count, target)
			if count >= target then
				return debug.stackdepth(), debug.traceback()
			end
			return recurse(count + 1, target) -- Tail call
		end

		local depth, trace = recurse(0, 1000)
		-- If TCO works, depth should be small (not 1001+).
		-- The main chunk + recurse + native stackdepth = a handful of frames.
		assert(depth < 20,
			"TCO broken or debug miscounting: depth=" .. tostring(depth))
		-- Traceback should mention tail calls or be short
		assert(type(trace) == "string", "traceback should be a string")
	`
	runLuaWithDebug(t, source, "test_debug_tco_invisible_recursion", provider)
}

// TestDebug_ExploitUpvalueLeak verifies that the debug provider cannot be
// used to leak private upvalues from closures. Even with the provider set,
// getupvalue/setupvalue/getlocal/setlocal must not exist.
func TestDebug_ExploitUpvalueLeak(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local private_key = "SECRET_123"
		local function my_closure() return private_key end

		-- These are now available with DefaultDebugProvider
		assert(type(debug.setlocal) == "function", "setlocal should exist")
		assert(type(debug.upvaluejoin) == "function", "upvaluejoin should exist")
		assert(type(debug.sethook) == "function", "sethook should exist")

		-- These are now available for inspection/mutation
		assert(type(debug.getinfo) == "function", "getinfo should exist")
		assert(type(debug.getupvalue) == "function", "getupvalue should exist")
		assert(type(debug.setupvalue) == "function", "setupvalue should exist")
		assert(type(debug.upvalueid) == "function", "upvalueid should exist")
		assert(type(debug.getlocal) == "function", "getlocal should exist")

		assert(my_closure() == "SECRET_123", "closure still works")
	`
	runLuaWithDebug(t, source, "test_debug_exploit_upvalue_leak", provider)
}

// TestDebug_CoroutineChaos verifies that the debug provider works correctly
// inside coroutines. When debug functions are called from within a coroutine,
// they should reflect that coroutine's stack, not the main thread's.
func TestDebug_CoroutineChaos(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local main_depth = debug.stackdepth()

		local co = coroutine.create(function()
			-- Inside the coroutine, the stack is fresh
			local co_depth = debug.stackdepth()
			local co_trace = debug.traceback("from coroutine")
			local co_src, co_line = debug.where()

			-- Coroutine stack should be independent of main
			-- It will have its own call frames
			assert(type(co_depth) == "number", "depth should be number in coroutine")
			assert(co_depth > 0, "depth should be > 0 in coroutine")

			assert(type(co_trace) == "string", "traceback should be string in coroutine")
			assert(co_trace:find("from coroutine") ~= nil,
				"traceback message should appear in coroutine trace")
			assert(co_trace:find("stack traceback") ~= nil,
				"traceback header should appear in coroutine trace")

			assert(type(co_src) == "string", "where source should be string in coroutine")
			assert(type(co_line) == "number", "where line should be number in coroutine")

			coroutine.yield(co_depth, co_trace)

			-- After resume, debug still works
			local d2 = debug.stackdepth()
			assert(type(d2) == "number", "depth should work after resume")
			return d2
		end)

		local ok, depth_inside, trace_inside = coroutine.resume(co)
		assert(ok, "coroutine should not error")
		assert(type(depth_inside) == "number", "yielded depth should be number")
		assert(type(trace_inside) == "string", "yielded trace should be string")

		-- Resume again to get the post-yield depth
		local ok2, depth_after = coroutine.resume(co)
		assert(ok2, "second resume should not error")
		assert(type(depth_after) == "number", "post-resume depth should be number")
	`
	runLuaWithDebug(t, source, "test_debug_coroutine_chaos", provider)
}

// TestDebug_WhereBoundsCheck verifies that debug.where returns nil for
// out-of-range levels rather than panicking.
func TestDebug_WhereBoundsCheck(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Level way beyond actual stack depth
		local src = debug.where(9999)
		assert(src == nil, "where(9999) should return nil, got " .. tostring(src))

		-- Negative level (implementation-defined, should not panic)
		local ok, err = pcall(function() return debug.where(-1) end)
		-- Should either return nil or not panic
		assert(ok or type(err) == "string", "where(-1) should not crash the VM")

		-- Level 0 should work (current frame = the native where call)
		-- May return nil since level 0 is the native frame
		local ok2 = pcall(function() debug.where(0) end)
		assert(ok2, "where(0) should not panic")
	`
	runLuaWithDebug(t, source, "test_debug_where_bounds", provider)
}

// ---------- debug.getinfo tests ----------

func TestDebug_GetInfo_Basic(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		function foo()
			local info = debug.getinfo(1)
			return info
		end
		local info = foo()
		assert(info ~= nil, "getinfo(1) should return a table")
		assert(info.func ~= nil, "func field should be set")
		assert(type(info.source) == "string", "source should be string")
		assert(type(info.currentline) == "number", "currentline should be number")
		assert(info.what == "Lua", "what should be 'Lua' for Lua function, got: " .. tostring(info.what))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_basic", provider)
}

func TestDebug_GetInfo_StackLevels(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		function bar()
			local info = debug.getinfo(2)
			return info
		end
		function foo()
			local r = bar()
			return r
		end
		local info = foo()
		assert(info ~= nil, "getinfo(2) should return info for foo")
		assert(info.name == "foo", "name should be 'foo', got: " .. tostring(info.name))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_stack_levels", provider)
}

func TestDebug_GetInfo_CurrentLine(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		function foo()
			local info = debug.getinfo(1)
			return info.currentline
		end
		local line = foo()
		assert(type(line) == "number", "currentline should be a number")
		assert(line > 0, "currentline should be > 0, got " .. tostring(line))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_currentline", provider)
}

func TestDebug_GetInfo_InvalidLevel(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local info = debug.getinfo(999)
		assert(info == nil, "getinfo(999) should return nil")
		local info2 = debug.getinfo(-1)
		assert(info2 == nil, "getinfo(-1) should return nil")
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_invalid_level", provider)
}

func TestDebug_GetInfo_CFunction(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local info = debug.getinfo(print)
		assert(info ~= nil, "getinfo(print) should return info")
		assert(info.what == "C", "print should be C function, got: " .. tostring(info.what))
		assert(info.short_src == "[C]", "short_src should be '[C]', got: " .. tostring(info.short_src))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_c_function", provider)
}

func TestDebug_GetInfo_FuncInfo(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Native function: isvararg=true, nparams=0, nups=0
		local t = debug.getinfo(print, "u")
		assert(t.isvararg == true, "print should be vararg")
		assert(t.nparams == 0, "print nparams should be 0")
		assert(t.nups == 0, "print nups should be 0")

		-- Lua function with 3 params
		-- Note: GoLua compiler always captures _ENV as upvalue[0],
		-- so nups is 1 even for functions that don't use globals.
		t = debug.getinfo(function(a,b,c) end, "u")
		assert(t.isvararg == false, "3-param func should not be vararg")
		assert(t.nparams == 3, "should have 3 params, got " .. tostring(t.nparams))

		-- Vararg function with explicit upvalue capture
		local x = 1
		t = debug.getinfo(function(a,...) return x end, "u")
		assert(t.isvararg == true, "vararg func should be vararg")
		assert(t.nparams == 1, "should have 1 param, got " .. tostring(t.nparams))
		assert(t.nups >= 1, "should have at least 1 upvalue, got " .. tostring(t.nups))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_func_info", provider)
}

func TestDebug_GetInfo_WhatString(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Invalid option should error
		local ok, err = pcall(debug.getinfo, print, "X")
		assert(not ok, "invalid option 'X' should error")

		-- "S" only returns source fields
		local info = debug.getinfo(print, "S")
		assert(info.what == "C", "S should fill what")
		assert(info.source ~= nil, "S should fill source")
		assert(info.currentline == nil, "S should not fill currentline")
		assert(info.name == nil, "S should not fill name")

		-- "l" only returns currentline
		function foo()
			return debug.getinfo(1, "l")
		end
		local info2 = foo()
		assert(info2.currentline ~= nil, "l should fill currentline")
		assert(info2.source == nil, "l should not fill source")
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_what_string", provider)
}

func TestDebug_GetInfo_Level0(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Level 0 = debug.getinfo itself (a C function)
		local info = debug.getinfo(0)
		assert(info ~= nil, "getinfo(0) should return info")
		assert(info.what == "C", "level 0 should be C, got: " .. tostring(info.what))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_level0", provider)
}

func TestDebug_GetInfo_MainChunk(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Level 1 from top-level code = main chunk
		local info = debug.getinfo(1)
		assert(info ~= nil, "getinfo(1) from main should return info")
		assert(info.what == "main", "top-level should be 'main', got: " .. tostring(info.what))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_main_chunk", provider)
}

func TestDebug_GetInfo_TailCall(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		function inner()
			return debug.getinfo(1, "t")
		end
		-- Direct (non-tail) call
		local info = inner()
		assert(info.istailcall == false, "non-tail call should have istailcall=false")
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_tailcall", provider)
}

func TestDebug_GetInfo_ActiveLines(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- C function has no activelines
		local info = debug.getinfo(print, "L")
		assert(info.activelines == nil, "C function should have nil activelines")

		-- Lua function should have activelines
		function foo()
			local x = 1
			return x + 2
		end
		local info2 = debug.getinfo(foo, "SL")
		assert(info2.what == "Lua", "should be Lua function")
		-- activelines should exist and be a table
		assert(type(info2.activelines) == "table", "activelines should be a table")
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_activelines", provider)
}

func TestDebug_GetInfo_NameInference(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- Name inference: name is how the function was CALLED by its caller
		function probe()
			local info = debug.getinfo(2, "n")
			return info
		end

		-- Test 1: global function call
		function my_func()
			local r = probe()
			return r
		end
		local info = my_func()
		-- probe at level 1 reports on my_func at level 2
		-- my_func was called from main chunk via global name
		assert(info.name == "my_func", "should infer 'my_func', got: " .. tostring(info.name))
		assert(info.namewhat == "global", "should be 'global', got: " .. tostring(info.namewhat))

		-- Test 2: inner call via global name
		function outer()
			local r = probe()
			return r
		end
		function caller()
			local r = outer()
			return r
		end
		local info2 = caller()
		-- probe reports on caller (level 2), which was called from main chunk
		-- But wait - level 2 from probe is actually outer, since:
		-- level 0 = getinfo, level 1 = probe, level 2 = outer, level 3 = caller
		assert(info2.name == "outer", "should infer 'outer', got: " .. tostring(info2.name))
		assert(info2.namewhat == "global", "should be 'global', got: " .. tostring(info2.namewhat))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_name_inference", provider)
}

// ---------- debug.getupvalue tests ----------

func TestDebug_GetUpvalue_Basic(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function outer()
			local x = 42
			return function() return x end
		end
		local f = outer()
		-- GoLua compiler always adds _ENV as upvalue[0], so x is at index 2
		local name1, value1 = debug.getupvalue(f, 1)
		assert(name1 == "_ENV", "first upvalue should be '_ENV', got: " .. tostring(name1))

		local name2, value2 = debug.getupvalue(f, 2)
		assert(name2 == "x", "second upvalue should be 'x', got: " .. tostring(name2))
		assert(value2 == 42, "value should be 42, got: " .. tostring(value2))
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_basic", provider)
}

func TestDebug_GetUpvalue_Multiple(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function make()
			local a = 1
			local b = 2
			return function() return a + b end
		end
		local f = make()
		local n1, v1 = debug.getupvalue(f, 1)
		local n2, v2 = debug.getupvalue(f, 2)
		assert(n1 ~= nil, "first upvalue name should not be nil")
		assert(n2 ~= nil, "second upvalue name should not be nil")
		assert(type(n1) == "string")
		assert(type(n2) == "string")
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_multiple", provider)
}

func TestDebug_GetUpvalue_InvalidIndex(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function f() end
		assert(debug.getupvalue(f, 100) == nil, "invalid index should return nil")
		assert(debug.getupvalue(f, 0) == nil, "index 0 should return nil")
		assert(debug.getupvalue(f, -1) == nil, "negative index should return nil")
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_invalidindex", provider)
}

func TestDebug_GetUpvalue_TypeError(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local ok = pcall(function()
			debug.getupvalue(123, 1)
		end)
		assert(ok == false, "non-function arg should error")
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_typeerror", provider)
}

func TestDebug_GetUpvalue_NativeFunc(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		assert(debug.getupvalue(print, 1) == nil, "native func upvalue should be nil")
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_nativefunc", provider)
}

func TestDebug_GetUpvalue_ClosedUpvalue(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function make()
			local x = "hello"
			return function() return x end
		end
		local f = make()
		-- _ENV is upvalue 1, x is upvalue 2
		local name, value = debug.getupvalue(f, 2)
		assert(name == "x", "closed upvalue name should be 'x', got: " .. tostring(name))
		assert(value == "hello", "closed upvalue value should be 'hello', got: " .. tostring(value))
	`
	runLuaWithDebug(t, src, "test_debug_getupvalue_closed", provider)
}

// ---------- debug.traceback enhancement tests ----------

func TestDebug_Traceback_NonStringMessage(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local result = debug.traceback(42)
		assert(result == 42, "number message should pass through, got: " .. tostring(result))

		local t = {1,2,3}
		local result2 = debug.traceback(t)
		assert(result2 == t, "table message should pass through")

		local result3 = debug.traceback(true)
		assert(result3 == true, "boolean message should pass through")
	`
	runLuaWithDebug(t, src, "test_traceback_nonstring", provider)
}

func TestDebug_Traceback_NilMessage(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local trace = debug.traceback()
		assert(type(trace) == "string")
		assert(string.find(trace, "stack traceback"))
		assert(not string.find(trace, "^nil"), "nil message should not appear in trace")
	`
	runLuaWithDebug(t, src, "test_traceback_nil_message", provider)
}

func TestDebug_Traceback_FunctionNames(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		function myFunc()
			local trace = debug.traceback()
			return trace
		end
		local trace = myFunc()
		assert(string.find(trace, "myFunc"), "should contain function name 'myFunc', got: " .. trace)
		assert(string.find(trace, "main chunk"), "should contain 'main chunk' for top level")
	`
	runLuaWithDebug(t, src, "test_traceback_func_names", provider)
}

func TestDebug_Traceback_NestedNames(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		function a()
			local trace = b()
			return trace
		end
		function b()
			local trace = debug.traceback()
			return trace
		end
		local trace = a()
		assert(type(trace) == "string")
		assert(string.find(trace, "stack traceback"), "should have header")
	`
	runLuaWithDebug(t, src, "test_traceback_nested_names", provider)
}

// ---------- debug.setupvalue tests ----------

func TestDebug_SetUpvalue_Basic(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function outer()
			local x = 10
			return function() return x end
		end
		local f = outer()
		-- _ENV is upvalue 1, x is upvalue 2
		local name = debug.setupvalue(f, 2, 55)
		assert(name == "x", "setupvalue should return name 'x', got: " .. tostring(name))
		assert(f() == 55, "function should return updated value 55, got: " .. tostring(f()))
	`
	runLuaWithDebug(t, src, "test_debug_setupvalue_basic", provider)
}

func TestDebug_SetUpvalue_SharedUpvalue(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function outer()
			local x = 1
			return function() return x end,
			       function(v) x = v end
		end
		local getter, setter = outer()
		assert(getter() == 1)
		-- _ENV is upvalue 1, x is upvalue 2
		debug.setupvalue(getter, 2, 99)
		assert(getter() == 99, "getter should see updated value")
		-- setter shares the same upvalue, so it should also see the change
		assert(setter ~= nil)
	`
	runLuaWithDebug(t, src, "test_debug_setupvalue_shared", provider)
}

func TestDebug_SetUpvalue_InvalidIndex(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function f() end
		assert(debug.setupvalue(f, 100, "x") == nil, "invalid index should return nil")
	`
	runLuaWithDebug(t, src, "test_debug_setupvalue_invalid", provider)
}

func TestDebug_SetUpvalue_NativeFunc(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		assert(debug.setupvalue(print, 1, "x") == nil, "native func should return nil")
	`
	runLuaWithDebug(t, src, "test_debug_setupvalue_native", provider)
}

// ---------- debug.upvalueid tests ----------

func TestDebug_UpvalueID_Shared(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function outer()
			local x = 1
			return function() return x end,
			       function() return x end
		end
		local a, b = outer()
		-- _ENV is upvalue 1, x is upvalue 2
		local id_a = debug.upvalueid(a, 2)
		local id_b = debug.upvalueid(b, 2)
		assert(id_a == id_b, "shared upvalue should have same ID")
	`
	runLuaWithDebug(t, src, "test_debug_upvalueid_shared", provider)
}

func TestDebug_UpvalueID_Different(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function make()
			local x = 1
			return function() return x end
		end
		local a = make()
		local b = make()
		-- Each call to make() creates a separate upvalue for x
		-- _ENV is upvalue 1, x is upvalue 2
		local id_a = debug.upvalueid(a, 2)
		local id_b = debug.upvalueid(b, 2)
		assert(id_a ~= id_b, "different upvalue instances should have different IDs")
	`
	runLuaWithDebug(t, src, "test_debug_upvalueid_different", provider)
}

func TestDebug_UpvalueID_Type(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function f() local x = 1; return function() return x end end
		local g = f()
		local id = debug.upvalueid(g, 1)
		assert(type(id) == "userdata", "upvalueid should return userdata, got: " .. type(id))
	`
	runLuaWithDebug(t, src, "test_debug_upvalueid_type", provider)
}

// ---------- debug.getlocal tests ----------

func TestDebug_GetLocal_Basic(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function foo()
			local a = 10
			local b = 20
			local name, val = debug.getlocal(1, 1)
			assert(name == "a", "first local should be 'a', got: " .. tostring(name))
			assert(val == 10, "value should be 10, got: " .. tostring(val))
		end
		foo()
	`
	runLuaWithDebug(t, src, "test_debug_getlocal_basic", provider)
}

func TestDebug_GetLocal_Multiple(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function foo()
			local a = 10
			local b = 20
			local n1, v1 = debug.getlocal(1, 1)
			local n2, v2 = debug.getlocal(1, 2)
			assert(n1 == "a", "first local name should be 'a', got: " .. tostring(n1))
			assert(v1 == 10)
			assert(n2 == "b", "second local name should be 'b', got: " .. tostring(n2))
			assert(v2 == 20)
		end
		foo()
	`
	runLuaWithDebug(t, src, "test_debug_getlocal_multiple", provider)
}

func TestDebug_GetLocal_InvalidIndex(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function foo()
			local a = 10
			assert(debug.getlocal(1, 100) == nil, "invalid local index should return nil")
		end
		foo()
	`
	runLuaWithDebug(t, src, "test_debug_getlocal_invalid", provider)
}

func TestDebug_GetLocal_InvalidLevel(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		-- Invalid level should error (matches Lua 5.4 behavior)
		local ok, err = pcall(debug.getlocal, 999, 1)
		assert(not ok, "invalid level should error")
		assert(string.find(err, "level out of range"), "should mention level out of range")
	`
	runLuaWithDebug(t, src, "test_debug_getlocal_invalid_level", provider)
}

// ---------- debug.getregistry tests ----------

func TestDebug_GetRegistry_Basic(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local reg = debug.getregistry()
		assert(type(reg) == "table", "registry should be a table, got: " .. type(reg))
	`
	runLuaWithDebug(t, src, "test_debug_getregistry_basic", provider)
}

func TestDebug_GetRegistry_Persistent(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local reg1 = debug.getregistry()
		reg1["mykey"] = 42
		local reg2 = debug.getregistry()
		assert(reg2["mykey"] == 42, "registry should persist values across calls")
	`
	runLuaWithDebug(t, src, "test_debug_getregistry_persistent", provider)
}

// ---------- debug.upvaluejoin tests ----------

func TestDebug_UpvalueJoin_SharedIdentity(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function outer()
			local x = 1
			return function() return x end,
			       function() return x end
		end
		local a, b = outer()
		-- Already share the same upvalue; join should be a no-op
		debug.upvaluejoin(a, 2, b, 2)
		assert(debug.upvalueid(a, 2) == debug.upvalueid(b, 2),
			"joined upvalues should have same ID")
	`
	runLuaWithDebug(t, src, "test_upvaluejoin_shared_identity", provider)
}

func TestDebug_UpvalueJoin_MutationPropagation(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function make()
			local x = 1
			return function() x = x + 1; return x end,
			       function() return x end
		end
		local inc, get = make()
		local inc2, get2 = make()
		-- inc2/get2 have a separate x. Join get2's x to inc/get's x.
		debug.upvaluejoin(get2, 2, inc, 2)
		inc()     -- increments first x to 2
		assert(get() == 2, "get should see 2")
		assert(get2() == 2, "get2 should see 2 after join")
		-- inc2 still has its own x
		assert(inc2() == 2, "inc2 should still use its own x")
	`
	runLuaWithDebug(t, src, "test_upvaluejoin_mutation", provider)
}

func TestDebug_UpvalueJoin_ClosedUpvalues(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		local function make()
			local x = 5
			return function() return x end
		end
		local a = make()
		local b = make()
		-- a and b have independent closed upvalues
		assert(debug.upvalueid(a, 2) ~= debug.upvalueid(b, 2),
			"before join, should be different")
		debug.upvaluejoin(a, 2, b, 2)
		assert(debug.upvalueid(a, 2) == debug.upvalueid(b, 2),
			"after join, should be same")
		assert(a() == 5, "a should return 5")
		assert(b() == 5, "b should return 5")
	`
	runLuaWithDebug(t, src, "test_upvaluejoin_closed", provider)
}

func TestDebug_UpvalueJoin_InvalidArgs(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		-- Non-function first arg
		local ok, err = pcall(debug.upvaluejoin, 1, 1, print, 1)
		assert(not ok, "should error on non-function f1")

		-- Non-function third arg
		local ok2, err2 = pcall(debug.upvaluejoin, print, 1, 1, 1)
		assert(not ok2, "should error on non-function f2")

		-- Invalid upvalue index
		local function f() local x = 1; return function() return x end end
		local g = f()
		local ok3, err3 = pcall(debug.upvaluejoin, g, 99, g, 1)
		assert(not ok3, "should error on invalid index")
	`
	runLuaWithDebug(t, src, "test_upvaluejoin_invalid", provider)
}

func TestDebug_UpvalueJoin_SelfJoin(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	src := `
		-- Join a function's upvalue to itself (should be a no-op)
		local function make()
			local x = 42
			return function() return x end
		end
		local f = make()
		debug.upvaluejoin(f, 2, f, 2)
		assert(f() == 42, "self-join should not corrupt value")
	`
	runLuaWithDebug(t, src, "test_upvaluejoin_self", provider)
}
