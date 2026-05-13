package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v1/vm"
)

func TestCoroutineCloseRegression_ClearsDebugIntrospectionState(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local co = coroutine.create(function()
			local x = 42
			coroutine.yield("pause")
		end)

		local ok, why = coroutine.resume(co)
		assert(ok and why == "pause")
		assert(debug.getinfo(co, 1, "Sl").what == "Lua")

		local cok, cerr = coroutine.close(co)
		assert(cok == true, tostring(cerr))
		assert(debug.getinfo(co, 1, "Sl") == nil)
		local ok, err = pcall(debug.getlocal, co, 1, 1)
		assert(ok == false)
		assert(tostring(err):find("level out of range", 1, true), tostring(err))
	`
	runLuaWithDebug(t, source, "test_coroutine_close_clears_debug_state", provider)
}
