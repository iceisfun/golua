package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

func TestDebugHookRegression_CallHookExposesTransferCounts(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local seen_f, seen_n
		local function foo(a, b)
			return a + b
		end
		debug.sethook(function(ev)
			if ev ~= "call" then return end
			local info = debug.getinfo(2, "frn")
			if info.func == foo then
				seen_f, seen_n = info.ftransfer, info.ntransfer
				debug.sethook()
			end
		end, "c")
		foo(7, 3)
		assert(seen_f == 1, tostring(seen_f))
		assert(seen_n == 2, tostring(seen_n))
	`
	runLuaWithDebug(t, source, "test_debug_call_hook_transfer_counts", provider)
}

func TestDebugHookRegression_LineHookTargetFrameKeepsEmptyNameWhat(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local seen_namewhat, seen_f, seen_n
		debug.sethook(function(ev, line)
			if ev ~= "line" then return end
			local info = debug.getinfo(2, "nr")
			seen_namewhat, seen_f, seen_n = info.namewhat, info.ftransfer, info.ntransfer
			debug.sethook()
		end, "l")
		local a = 0
		a = a + 1
		assert(seen_namewhat == "", tostring(seen_namewhat))
		assert(seen_f == 0, tostring(seen_f))
		assert(seen_n == 0, tostring(seen_n))
	`
	runLuaWithDebug(t, source, "test_debug_line_hook_namewhat", provider)
}

// Errors raised inside a debug hook are ordinary errors in reference Lua:
// luaD_hook calls the hook like any other function, so the nearest
// pcall/xpcall around the hooked code catches them. golua used to box them in
// a sentinel that every pcall re-raised, killing the whole script.
func TestDebugHookRegression_HookErrorCaughtByPcall(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		-- line hook, pcall
		local ok, e = pcall(function()
			debug.sethook(function() debug.sethook() error("lineerr") end, "l")
			local a = 1
			return a
		end)
		assert(ok == false, "line hook error not caught")
		assert(e:find("lineerr", 1, true), tostring(e))

		-- count hook, pcall; the hook stays installed and keeps erroring
		ok, e = pcall(function()
			debug.sethook(function() error("counterr") end, "", 100)
			local s = 0
			for i = 1, 10000 do s = s + i end
			return s
		end)
		debug.sethook()
		assert(ok == false, "count hook error not caught")
		assert(e:find("counterr", 1, true), tostring(e))

		-- call hook, xpcall: the message handler sees the hook's error
		ok, e = xpcall(function()
			debug.sethook(function() debug.sethook() error("callerr") end, "c")
			math.abs(1)
		end, function(m) return "H:" .. m end)
		assert(ok == false, "call hook error not caught")
		assert(e:find("^H:.*callerr"), tostring(e))

		-- non-string error values pass through unchanged
		local obj = {}
		ok, e = pcall(function()
			debug.sethook(function() debug.sethook() error(obj) end, "l")
			local a = 1
			return a
		end)
		assert(ok == false and e == obj, "hook error object changed")

		-- a library error in the hook gets its position like any other
		ok, e = pcall(function()
			debug.sethook(function() debug.sethook() math.floor("x") end, "l")
			local a = 1
			return a
		end)
		assert(ok == false and e:find(":%d+: bad argument #1 to 'floor'"), tostring(e))
	`
	runLuaWithDebug(t, source, "test_debug_hook_error_caught", provider)
}
