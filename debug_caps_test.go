package golua_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// capProvider is a debug provider that returns a specific LuaDebugCaps.
type capProvider struct{ caps vm.LuaDebugCaps }

func (p *capProvider) Capabilities(context.Context) vm.LuaDebugCaps { return p.caps }

// allCaps returns a LuaDebugCaps with every field set to true.
func allCaps() vm.LuaDebugCaps {
	return vm.LuaDebugCaps{
		AllowTraceback:      true,
		AllowStackDepth:     true,
		AllowWhere:          true,
		AllowGetInfo:        true,
		AllowGetUpvalue:     true,
		AllowSetUpvalue:     true,
		AllowUpvalueID:      true,
		AllowGetLocal:       true,
		AllowSetLocal:       true,
		AllowGetRegistry:    true,
		AllowGetMetatable:   true,
		AllowSetMetatable:   true,
		AllowSetHook:        true,
		AllowGetHook:        true,
		AllowUpvalueJoin:  true,
		AllowGetUserValue: true,
		AllowSetUserValue:   true,
	}
}

// capFieldToFunc maps each LuaDebugCaps field name to the debug.* function
// it gates and a Lua snippet that exercises it without error.
var capFieldToFunc = []struct {
	field    string // LuaDebugCaps field name
	luaName  string // debug.X function name
	exercise string // Lua snippet that calls the function successfully
}{
	{"AllowTraceback", "traceback", `debug.traceback()`},
	{"AllowStackDepth", "stackdepth", `debug.stackdepth()`},
	{"AllowWhere", "where", `debug.where()`},
	{"AllowGetInfo", "getinfo", `debug.getinfo(1)`},
	{"AllowGetUpvalue", "getupvalue", `local f = function() end; debug.getupvalue(f, 1)`},
	{"AllowSetUpvalue", "setupvalue", `local x = 1; local f = function() return x end; debug.setupvalue(f, 1, 2)`},
	{"AllowUpvalueID", "upvalueid", `local x = 1; local f = function() return x end; debug.upvalueid(f, 1)`},
	{"AllowGetLocal", "getlocal", `debug.getlocal(1, 1)`},
	{"AllowSetLocal", "setlocal", `debug.setlocal(1, 1, nil)`},
	{"AllowGetRegistry", "getregistry", `debug.getregistry()`},
	{"AllowGetMetatable", "getmetatable", `debug.getmetatable({})`},
	{"AllowSetMetatable", "setmetatable", `debug.setmetatable({}, nil)`},
	{"AllowSetHook", "sethook", `debug.sethook(function() end, "")`},
	{"AllowGetHook", "gethook", `debug.gethook()`},
	{"AllowUpvalueJoin", "upvaluejoin", `local x,y=1,2; local f=function()return x end; local g=function()return y end; debug.upvaluejoin(f,1,g,1)`},
	{"AllowGetUserValue", "getuservalue", `debug.getuservalue({})`},
	{"AllowSetUserValue", "setuservalue", `pcall(debug.setuservalue, {}, nil)`},
}

// setCap sets a single bool field on a LuaDebugCaps by name.
func setCap(caps *vm.LuaDebugCaps, field string, val bool) {
	rv := reflect.ValueOf(caps).Elem()
	rv.FieldByName(field).SetBool(val)
}

// newDebugVM creates a VM with the given debug caps and stdlib loaded.
func newDebugVM(t *testing.T, caps vm.LuaDebugCaps) *vm.VM {
	t.Helper()
	v := vm.New()
	if err := v.SetDebugProvider(&capProvider{caps}); err != nil {
		t.Fatalf("SetDebugProvider: %v", err)
	}
	stdlib.Open(v)
	return v
}

// runSnippet compiles and runs a Lua snippet, returning any error.
func runSnippet(v *vm.VM, source string) error {
	block, err := parser.Parse("test", source)
	if err != nil {
		return err
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		return err
	}
	_, err = v.Run(proto)
	return err
}

// TestDebugCaps_EnabledAlone verifies that each cap, when it is the ONLY one
// enabled, exposes its function and the function is callable.
func TestDebugCaps_EnabledAlone(t *testing.T) {
	for _, tc := range capFieldToFunc {
		t.Run(tc.field, func(t *testing.T) {
			// Only this cap is true.
			caps := vm.LuaDebugCaps{}
			setCap(&caps, tc.field, true)
			v := newDebugVM(t, caps)

			// The function should exist.
			check := `assert(type(debug.` + tc.luaName + `) == "function", "debug.` + tc.luaName + ` should be a function")`
			if err := runSnippet(v, check); err != nil {
				t.Fatalf("expected debug.%s to be a function: %v", tc.luaName, err)
			}

			// The function should be callable.
			if err := runSnippet(v, tc.exercise); err != nil {
				t.Fatalf("debug.%s should be callable: %v", tc.luaName, err)
			}
		})
	}
}

// TestDebugCaps_DisabledAlone verifies that each cap, when it is the ONLY one
// disabled (all others true), hides its function while the rest remain.
func TestDebugCaps_DisabledAlone(t *testing.T) {
	for _, tc := range capFieldToFunc {
		t.Run(tc.field, func(t *testing.T) {
			// All caps true except this one.
			caps := allCaps()
			setCap(&caps, tc.field, false)
			v := newDebugVM(t, caps)

			// The function should be nil.
			check := `assert(debug.` + tc.luaName + ` == nil, "debug.` + tc.luaName + ` should be nil when disabled")`
			if err := runSnippet(v, check); err != nil {
				t.Fatalf("expected debug.%s to be nil when disabled: %v", tc.luaName, err)
			}

			// All OTHER functions should still be present.
			for _, other := range capFieldToFunc {
				if other.field == tc.field {
					continue
				}
				otherCheck := `assert(type(debug.` + other.luaName + `) == "function", "debug.` + other.luaName + ` should still be a function")`
				if err := runSnippet(v, otherCheck); err != nil {
					t.Fatalf("disabling %s should not affect debug.%s: %v", tc.field, other.luaName, err)
				}
			}
		})
	}
}

// TestDebugCaps_AllDisabled verifies that with all caps false, the debug table
// exists but contains no functions.
func TestDebugCaps_AllDisabled(t *testing.T) {
	v := newDebugVM(t, vm.LuaDebugCaps{})

	for _, tc := range capFieldToFunc {
		check := `assert(debug.` + tc.luaName + ` == nil, "debug.` + tc.luaName + ` should be nil")`
		if err := runSnippet(v, check); err != nil {
			t.Fatalf("with all caps disabled, debug.%s should be nil: %v", tc.luaName, err)
		}
	}
}

// TestDebugCaps_AllEnabled verifies that with all caps true, every function
// is present and callable.
func TestDebugCaps_AllEnabled(t *testing.T) {
	v := newDebugVM(t, allCaps())

	for _, tc := range capFieldToFunc {
		check := `assert(type(debug.` + tc.luaName + `) == "function", "debug.` + tc.luaName + ` should be a function")`
		if err := runSnippet(v, check); err != nil {
			t.Fatalf("with all caps enabled, debug.%s should be a function: %v", tc.luaName, err)
		}
		if err := runSnippet(v, tc.exercise); err != nil {
			t.Fatalf("with all caps enabled, debug.%s should be callable: %v", tc.luaName, err)
		}
	}
}
