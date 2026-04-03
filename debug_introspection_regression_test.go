package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

func TestDebugGetInfoRegression_ActiveLinesIncludeLoopHeaders(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local function loopy(n)
  local sum = 0
  while n > 0 do
    sum = sum + n
    n = n - 1
  end
  for i = 1, 2 do
    sum = sum + i
  end
  return sum
end

local lines = debug.getinfo(loopy, "L").activelines
assert(lines[4], "while body line missing")
assert(lines[5], "while update line missing")
assert(lines[8], "for body line missing")
assert(lines[3], "while header should be active (has TEST instruction)")
assert(lines[7], "for header should be active (has FORPREP instruction)")
`
	runLuaWithDebug(t, source, "test_debug_activelines_loop_headers", provider)
}

func TestDebugGetInfoRegression_ReturnHooksExposeTransferCounts(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local seen_lua_f, seen_lua_n
local seen_c_f, seen_c_n

local function foo()
  return 10, 20
end

debug.sethook(function(event)
  if event ~= "return" then
    return
  end

  local info = debug.getinfo(2, "frS")
  if info.func == foo then
    seen_lua_f, seen_lua_n = info.ftransfer, info.ntransfer
  elseif info.func == pcall then
    seen_c_f, seen_c_n = info.ftransfer, info.ntransfer
  end
end, "r")

local ok, value = pcall(function()
  return "pcall-result"
end)
assert(ok and value == "pcall-result")

local a, b = foo()
debug.sethook()

assert(a == 10 and b == 20)
assert(type(seen_lua_f) == "number" and seen_lua_f > 0, tostring(seen_lua_f))
assert(seen_lua_n == 2, tostring(seen_lua_n))
assert(type(seen_c_f) == "number" and seen_c_f > 0, tostring(seen_c_f))
assert(type(seen_c_n) == "number" and seen_c_n > 0, tostring(seen_c_n))
`
	runLuaWithDebug(t, source, "test_debug_return_hook_transfer_counts", provider)
}

func TestDebugGetLocalRegression_LuaFramesExposeTemporaries(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local seen_name, seen_value

local function foo(a)
  local x = a + 1
  return x, a * 2
end

debug.sethook(function(event)
  if event ~= "return" then
    return
  end

  local info = debug.getinfo(2, "f")
  if info.func == foo then
    seen_name, seen_value = debug.getlocal(2, 3)
  end
end, "r")

local a, b = foo(4)
debug.sethook()

assert(a == 5 and b == 8)
assert(seen_name == "(temporary)", tostring(seen_name))
assert(seen_value == 5 or seen_value == 8, tostring(seen_value))
`
	runLuaWithDebug(t, source, "test_debug_getlocal_lua_temporaries", provider)
}

func TestDebugGetLocalRegression_NativeReturnFramesExposeTemporaries(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local seen = {}

debug.sethook(function(event)
  if event ~= "return" then
    return
  end

  local info = debug.getinfo(2, "f")
  if info.func == pcall then
    seen[1], seen[2] = debug.getlocal(2, 1)
    seen[3], seen[4] = debug.getlocal(2, 2)
    seen[5], seen[6] = debug.getlocal(2, 3)
  end
end, "r")

local ok, value = pcall(function()
  return "result"
end)
debug.sethook()

assert(ok and value == "result")
assert(seen[1] == "(C temporary)", tostring(seen[1]))
assert(seen[3] == "(C temporary)", tostring(seen[3]))
assert(seen[5] == "(C temporary)", tostring(seen[5]))
assert(seen[6] == "result", tostring(seen[6]))
`
	runLuaWithDebug(t, source, "test_debug_getlocal_native_temporaries", provider)
}

func TestDebugCoroutineErrorSnapshotRegression_GetInfoGetLocalSetLocal(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local co = coroutine.create(function()
  local x = 42
  error("boom")
end)

local ok, err = coroutine.resume(co)
assert(ok == false, "resume should fail")
assert(tostring(err):find("boom", 1, true), tostring(err))

local info = debug.getinfo(co, 1, "Slun")
assert(type(info) == "table", tostring(info))
assert(info.what == "Lua", tostring(info.what))
assert(info.currentline == 3, tostring(info.currentline))

local name, val = debug.getlocal(co, 1, 1)
assert(name == "x", tostring(name))
assert(val == 42, tostring(val))

local setname = debug.setlocal(co, 1, 1, 99)
assert(setname == "x", tostring(setname))

local name2, val2 = debug.getlocal(co, 1, 1)
assert(name2 == "x", tostring(name2))
assert(val2 == 99, tostring(val2))
`
	runLuaWithDebug(t, source, "test_debug_dead_coroutine_error_snapshot", provider)
}

func TestDebugSetLocalRegression_CoroutineLevelCheckedBeforeMissingValue(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local co = coroutine.create(function() end)
		coroutine.resume(co)
		local ok, err = pcall(debug.setlocal, co, 1, 1)
		assert(ok == false)
		assert(tostring(err):find("bad argument #2 to 'debug.setlocal' (level out of range)", 1, true), tostring(err))
	`
	runLuaWithDebug(t, source, "test_debug_setlocal_thread_level_error_order", provider)
}

func TestDebugGetInfoRegression_ActiveLinesIncludeIfHeaders(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function f(a)
			if a then
				return 1
			else
				return 2
			end
		end

		local lines = debug.getinfo(f, "L").activelines
		assert(lines[3], "if header should be active")
		assert(lines[4], "then branch line missing")
		assert(lines[6], "else branch line missing")
		assert(lines[8], "function end line missing")
	`
	runLuaWithDebug(t, source, "test_debug_activelines_if_header", provider)
}

func TestDebugGetLocalRegression_SuspendedCoroutineHidesLuaTemporaries(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `local co = coroutine.create(function(a)
  local x = a + 1
  coroutine.yield("pause")
  return x
end)

local ok, why = coroutine.resume(co, 4)
assert(ok and why == "pause")

local name, val = debug.getlocal(co, 1, 1)
assert(name == "a", tostring(name))
assert(val == 4, tostring(val))

assert(select('#', debug.getlocal(co, 1, 3)) == 1)
assert(debug.getlocal(co, 1, 3) == nil)
`
	runLuaWithDebug(t, source, "test_debug_getlocal_suspended_coroutine_temporaries", provider)
}
