package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

func TestDebugStdlibRegression_SetCStackLimitRemovedInLua55(t *testing.T) {
	// Lua 5.5: debug.setcstacklimit was removed.
	provider := vm.NewDefaultDebugProvider()
	source := `
		assert(debug.setcstacklimit == nil, "debug.setcstacklimit should be nil in 5.5")
	`
	runLuaWithDebug(t, source, "test_debug_setcstacklimit_removed", provider)
}

func TestDebugStdlibRegression_GetInfoReportsInvalidOptionCharacter(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local ok, err = pcall(debug.getinfo, print, ">S")
		assert(ok == false)
		assert(tostring(err):find("invalid option", 1, true), tostring(err))
	`
	runLuaWithDebug(t, source, "test_debug_getinfo_invalid_option_char", provider)
}

func TestDebugStdlibRegression_GetRegistryExposesPackageAliases(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local r = debug.getregistry()
		assert(r._LOADED == package.loaded)
		assert(r._PRELOAD == package.preload)
	`
	runLuaWithDebug(t, source, "test_debug_getregistry_package_aliases", provider)
}
