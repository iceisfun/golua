package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/vm"
)

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
