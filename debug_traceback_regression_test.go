package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
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
  assert(tb:find("[C]: in global 'pcall'", 1, true), tb)
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
assert(tb:find("global 'error'", 1, true), tb)
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
assert(msg:find("[C]: in field 'traceback'", 1, true), msg)
`
	runLuaWithDebug(t, source, "test_debug_traceback_qualified_c_name", provider)
}

func TestDebugTracebackRegression_TopLevelNativeFramesResolveDisplayName(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local cases = {
  {fn = function() return math.abs() end, want = "[C]: in field 'abs'"},
  {fn = function() return string.byte() end, want = "[C]: in field 'byte'"},
  {fn = function() return table.insert({}) end, want = "[C]: in field 'insert'"},
  {fn = function() return pcall() end, want = "[C]: in global 'pcall'"},
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

// The host [C] frame below the main chunk is a level of its own: a traceback
// starting there prints it, and one starting past it prints nothing.
func TestDebugTracebackRegression_LevelAtHostFrame(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `assert(debug.traceback("m", 2) == "m\nstack traceback:\n\t[C]: in ?")
assert(debug.traceback("m", 3) == "m\nstack traceback:")
local function f() return debug.traceback("f", 3) end
assert(f() == "f\nstack traceback:\n\t[C]: in ?")
`
	runLuaWithDebug(t, source, "test_debug_traceback_level_at_host_frame", provider)
}

// Reference luaL_traceback prints a trace of up to 22 levels whole, and on
// longer ones reports one fewer skipped level than it omits. golua truncated
// a 22-level trace and reported the true count; print what reference prints.
func TestDebugTracebackRegression_SkipCountMatchesReference(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function deep(n)
  if n == 0 then return debug.traceback("d") end
  return (deep(n - 1))
end
local function levels(tb)
  local n = 0
  for _ in tb:gmatch("\n\t[^.\n]") do n = n + 1 end
  return n, tb:match("skipping (%d+) levels")
end
-- traceback level 1 is deep(0); with the main chunk and the host frame
-- below, deep(n) yields n + 3 levels.
local n, skip = levels(deep(19))
assert(n == 22 and skip == nil, n .. " " .. tostring(skip))
n, skip = levels(deep(20))
assert(n == 21 and skip == "1", n .. " " .. tostring(skip))
n, skip = levels(deep(30))
assert(n == 21 and skip == "11", n .. " " .. tostring(skip))
`
	runLuaWithDebug(t, source, "test_debug_traceback_skip_count", provider)
}

// A __close handler run by a normal block exit is called from the closing
// Lua frame, so reference names it "metamethod 'close'" at every depth.
// golua dropped the name whenever the traceback started in a deeper Lua
// frame, a heuristic meant only for handlers run while an error unwinds.
func TestDebugTracebackRegression_CloseHandlerNamedBelowDeeperFrame(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function tb(m) return debug.traceback(m) end
local seen = {}
local mt = {__close = function() seen[#seen + 1] = tb("x") end}
do local c <close> = setmetatable({}, mt) end
local function r() local c <close> = setmetatable({}, mt) return 1 end
r()
pcall(function() local c <close> = setmetatable({}, mt) error("e") end)
assert(seen[1]:find("\n\t[^\n]*:3: in metamethod 'close'\n"), seen[1])
assert(seen[2]:find("\n\t[^\n]*:3: in metamethod 'close'\n"), seen[2])
-- closed while the error unwinds: called from pcall, so no name
assert(seen[3]:find("\n\t[^\n]*:3: in function <"), seen[3])
`
	runLuaWithDebug(t, source, "test_debug_traceback_close_handler_deeper_frame", provider)
}

// A tail call reuses the caller's frame, and reference funcnamefromcall
// gives a CIST_TAIL frame no name. golua's traceback still labelled the
// tail-called function with the metamethod name of the frame it replaced.
func TestDebugTracebackRegression_TailCalledFrameDropsMetamethodName(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function tb(m) return debug.traceback(m) end
local i = setmetatable({}, {__index = function() return tb("idx") end})
local a = setmetatable({}, {__add = function() return tb("add") end})
for _, msg in ipairs({i.x, a + 1}) do
  assert(msg:find("\n\t[^\n]*:1: in function <[^\n]*:1>\n\t%(%.%.%.tail calls%.%.%.%)\n"), msg)
  assert(not msg:find("in metamethod", 1, true), msg)
end
`
	runLuaWithDebug(t, source, "test_debug_traceback_tail_called_metamethod", provider)
}
