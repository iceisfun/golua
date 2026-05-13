package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v1/vm"
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
