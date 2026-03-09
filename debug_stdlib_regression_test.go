package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/vm"
)

func TestDebugStdlibRegression_SetCStackLimitExistsAndReturnsPreviousLimit(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		assert(type(debug.setcstacklimit) == "function")
		local a = debug.setcstacklimit(100)
		local b = debug.setcstacklimit(200)
		local ok1, c = pcall(debug.setcstacklimit, -1)
		local ok2, d = pcall(debug.setcstacklimit, 0)
		assert(a == 200, tostring(a))
		assert(b == 200, tostring(b))
		assert(ok1 and c == 200, tostring(c))
		assert(ok2 and d == 200, tostring(d))
	`
	runLuaWithDebug(t, source, "test_debug_setcstacklimit_exists", provider)
}
