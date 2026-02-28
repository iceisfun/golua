// Package stdlib implements the Lua 5.4 standard library for the GoLua VM.
//
// Call [Open] to register all standard library functions in a VM. The
// following modules are always registered: basic functions (print, assert,
// type, etc.), string, math, table, coroutine, load, bit32, and utf8.
//
// Conditional modules require a provider to be set on the VM before calling
// Open:
//   - io: requires [vm.LuaIoProvider] (e.g. [vm.JailedIoProvider])
//   - os: requires [vm.LuaOsProvider] (e.g. [vm.DefaultOsProvider])
//   - debug: requires [vm.LuaDebugProvider] (e.g. [vm.DefaultDebugProvider])
//   - chan: requires [vm.LuaChanProvider] (e.g. [vm.DefaultChanProvider])
//   - time: requires [vm.LuaTimeProvider] (e.g. [vm.DefaultTimeProvider])
//
// GoLua extensions beyond Lua 5.4: bit32 (from Lua 5.2), chan (Go channels),
// time (millisecond timing), glob (Go-style pattern matching).
//
// Lua 5.4 Reference: §6 (standard libraries).
package stdlib

import (
	"github.com/iceisfun/golua/vm"
)

// Open registers all standard library functions and modules in the VM.
// Conditional modules (io, os, debug, chan, time) are only registered if
// their corresponding provider has been set on the VM. Open also sets
// _G and _VERSION ("Lua 5.4") as globals.
func Open(v *vm.VM) {
	// Basic functions
	v.SetGlobal("print", vm.NewNativeFunc(luaPrint))
	v.SetGlobal("assert", vm.NewNativeFunc(luaAssert))
	v.SetGlobal("type", vm.NewNativeFunc(luaType))
	v.SetGlobal("tostring", vm.NewNativeFunc(luaToString))
	v.SetGlobal("tonumber", vm.NewNativeFunc(luaToNumber))
	v.SetGlobal("error", vm.NewNativeFunc(luaError))
	v.SetGlobal("pcall", vm.NewNativeFunc(luaPcall))
	v.SetGlobal("xpcall", vm.NewNativeFunc(luaXpcall))
	v.SetGlobal("pairs", vm.NewNativeFunc(luaPairs))
	v.SetGlobal("ipairs", vm.NewNativeFunc(luaIpairs))
	v.SetGlobal("next", vm.NewNativeFunc(luaNext))
	v.SetGlobal("select", vm.NewNativeFunc(luaSelect))
	v.SetGlobal("rawget", vm.NewNativeFunc(luaRawget))
	v.SetGlobal("rawset", vm.NewNativeFunc(luaRawset))
	v.SetGlobal("rawequal", vm.NewNativeFunc(luaRawequal))
	v.SetGlobal("rawlen", vm.NewNativeFunc(luaRawlen))
	v.SetGlobal("getmetatable", vm.NewNativeFunc(luaGetmetatable))
	v.SetGlobal("setmetatable", vm.NewNativeFunc(luaSetmetatable))
	v.SetGlobal("collectgarbage", vm.NewNativeFunc(luaCollectgarbage))
	v.SetGlobal("warn", vm.NewNativeFunc(luaWarn))

	// Output inspection (for testing)
	v.SetGlobal("_lastoutput", vm.NewNativeFunc(luaLastOutput))
	v.SetGlobal("_outputlines", vm.NewNativeFunc(luaOutputLines))

	// _G points to the globals table
	v.SetGlobal("_G", vm.NewTable(v.Globals()))

	// _VERSION
	v.SetGlobal("_VERSION", vm.NewString("Lua 5.4"))

	// String library
	openString(v)

	// Math library
	openMath(v)

	// Table library
	openTable(v)

	// Coroutine library
	openCoroutine(v)

	// Load functions (load, loadfile, dofile)
	openLoad(v)

	// IO library (only if IoProvider is set)
	openIo(v)

	// OS library (only if OsProvider is set)
	openOs(v)

	// Debug library (only if DebugProvider is set)
	openDebug(v)

	// Channel library (only if ChanProvider is set)
	openChan(v)

	// Bit32 library (Lua 5.2 compat)
	openBit32(v)

	// UTF-8 library (strict mode only)
	openUtf8(v)

	// Glob library (Go-style pattern matching)
	openGlob(v)

	// Time library (only if TimeProvider is set)
	openTime(v)
}
