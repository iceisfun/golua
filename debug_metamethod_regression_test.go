package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/vm"
)

func TestDebugMetamethodRegression_CloseGetInfoNameIsNil(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local seen_name, seen_namewhat

		local function run()
			local obj = setmetatable({}, {__close = function()
				local info = debug.getinfo(1, "n")
				seen_name, seen_namewhat = info.name, info.namewhat
				error("close")
			end})
			local x <close> = obj
			error("body")
		end

		xpcall(run, function(e) return e end)
		assert(seen_name == nil, tostring(seen_name))
		assert(seen_namewhat == "", tostring(seen_namewhat))
	`
	runLuaWithDebug(t, source, "test_debug_close_getinfo_name_nil", provider)
}
