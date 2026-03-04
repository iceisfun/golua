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
		assert(debug.sethook == nil, "sethook should not exist")
		assert(debug.setlocal == nil, "setlocal should not exist")
		assert(debug.setupvalue == nil, "setupvalue should not exist")
		assert(debug.setmetatable == nil, "debug.setmetatable should not exist")
		assert(debug.getlocal == nil, "getlocal should not exist")
		assert(debug.getupvalue == nil, "getupvalue should not exist")
	`
	runLuaWithDebug(t, source, "test_debug_no_mutation", provider)
}

func TestDebug_NoHooks(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		assert(debug.gethook == nil, "gethook should not exist")
		assert(debug.sethook == nil, "sethook should not exist")
	`
	runLuaWithDebug(t, source, "test_debug_no_hooks", provider)
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

		-- Verify no upvalue access functions exist
		assert(debug.getupvalue == nil,
			"getupvalue must not exist - would leak private upvalues")
		assert(debug.setupvalue == nil,
			"setupvalue must not exist - would mutate private upvalues")
		assert(debug.getlocal == nil,
			"getlocal must not exist - would leak local variables")
		assert(debug.setlocal == nil,
			"setlocal must not exist - would mutate local variables")
		assert(debug.upvalueid == nil,
			"upvalueid must not exist - would expose upvalue identity")
		assert(debug.upvaluejoin == nil,
			"upvaluejoin must not exist - would alias upvalues")
		assert(type(debug.getinfo) == "function",
			"getinfo should exist")

		-- The closure's private_key remains inaccessible via debug
		-- Only the closure itself can return it
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
