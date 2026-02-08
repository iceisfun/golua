package main

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
		assert(debug.getinfo == nil, "getinfo should not exist")
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
