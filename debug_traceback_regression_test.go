package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithDebugError(t *testing.T, source, name string, provider vm.LuaDebugProvider) string {
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
	if err == nil {
		t.Fatal("expected runtime error")
	}
	return err.Error()
}

func TestDebugTracebackRegression_BottomCFrame(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function capture()
  return debug.traceback("", 0)
end

local tb = capture()
assert(tb:find("stack traceback:", 1, true), tb)
assert(tb:find("[C]: in ?", 1, true), tb)
`
	runLuaWithDebug(t, source, "test_debug_traceback_bottom_c_frame", provider)
}

func TestDebugTracebackRegression_StrippedDumpOmitsZeroLine(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	// A function reloaded from a debug-stripped dump has no line table, so its
	// frame's current line is unknown. Reference luaL_traceback then prints
	// "source: in ..." (no ":line"), NOT "source:0: in ...".
	source := `local g = load(string.dump(function()
  return debug.traceback("msg", 1)
end, true))
local tb = g()
assert(tb:find("?: in local 'g'", 1, true), tb)
assert(not tb:find("?:0:", 1, true), tb)
`
	runLuaWithDebug(t, source, "test_debug_traceback_stripped_dump", provider)
}

func TestDebugTracebackRegression_NativeFramesUseCFormatting(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function capture()
  local ok, tb = pcall(function()
    return debug.traceback("", 0)
  end)
  assert(ok, tb)
  assert(tb:find("[C]: in function 'pcall'", 1, true), tb)
end

capture()
`
	runLuaWithDebug(t, source, "test_debug_traceback_native_frame_format", provider)
}

func TestDebugTracebackRegression_TailCallTopFrameUsesFunctionLocation(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function target()
  return debug.traceback("", 0)
end

local function caller()
  return target()
end

local tb = caller()
assert(tb:find("in function <", 1, true), tb)
assert(not tb:find("in function 'target'", 1, true), tb)
`
	runLuaWithDebug(t, source, "test_debug_traceback_tailcall_top_frame", provider)
}

func TestDebugTracebackRegression_NegativeLevelDoesNotCrash(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local ok, tb = pcall(debug.traceback, "msg", -1)
assert(ok, tb)
assert(type(tb) == "string", type(tb))
assert(tb:find("stack traceback:", 1, true), tb)
`
	runLuaWithDebug(t, source, "test_debug_traceback_negative_level", provider)
}

func TestDebugTracebackRegression_ErroredCoroutinePreservesStack(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local co = coroutine.create(function()
  local function boom()
    error("boom")
  end

  boom()
end)

local ok, err = coroutine.resume(co)
assert(ok == false, "resume should fail")
assert(type(err) == "string", err)

local tb = debug.traceback(co)
assert(tb:find("stack traceback:", 1, true), tb)
assert(tb:find("boom", 1, true), tb)
assert(tb:find("function 'error'", 1, true), tb)
	`
	runLuaWithDebug(t, source, "test_debug_traceback_errored_coroutine", provider)
}

func TestDebugTracebackRegression_QualifiedDebugTracebackCName(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function run()
  return xpcall(function()
    error("body")
  end, function(e)
    return debug.traceback(e, 0)
  end)
end

local ok, msg = run()
assert(ok == false, tostring(ok))
assert(msg:find("[C]: in function 'debug.traceback'", 1, true), msg)
`
	runLuaWithDebug(t, source, "test_debug_traceback_qualified_c_name", provider)
}

func TestDebugTracebackRegression_TopLevelNativeFramesResolveDisplayName(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local cases = {
  {fn = function() return math.abs() end, want = "[C]: in function 'math.abs'"},
  {fn = function() return string.byte() end, want = "[C]: in function 'string.byte'"},
  {fn = function() return table.insert({}) end, want = "[C]: in function 'table.insert'"},
  {fn = function() return pcall() end, want = "[C]: in function 'pcall'"},
}

for _, tc in ipairs(cases) do
  local ok, err = xpcall(tc.fn, debug.traceback)
  assert(ok == false)
  local msg = tostring(err)
  assert(msg:find(tc.want, 1, true), msg)
end
`
	runLuaWithDebug(t, source, "test_debug_traceback_top_level_native_names", provider)
}

func TestDebugTracebackRegression_HookFramesUseHookName(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		debug.sethook(function()
			error(debug.traceback("hookboom", 0))
		end, "c")
		pcall(function() end)
	`
	err := runLuaWithDebugError(t, source, "test_debug_traceback_hook_name", provider)
	if want := "in hook '?'"; !strings.Contains(err, want) {
		t.Fatalf("expected %q in error, got: %s", want, err)
	}
}

func TestDebugTracebackRegression_XPCallCloseErrorUsesFunctionFrameInLuaHandler(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function run()
			local obj = setmetatable({}, {__close = function()
				error("close")
			end})
			local x <close> = obj
			error("body")
		end

		local ok, msg = xpcall(run, function(e)
			return debug.traceback(e, 0)
		end)
		assert(ok == false)
		assert(msg:find("in function <", 1, true), msg)
		assert(msg:find("in metamethod 'close'", 1, true) == nil, msg)
	`
	runLuaWithDebug(t, source, "test_debug_traceback_xpcall_close_custom_handler", provider)
}
