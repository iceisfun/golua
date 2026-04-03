package stdlib

import (
	"context"
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// testOsProvider is a minimal LuaOsProvider for testing env var lookups.
type testOsProvider struct {
	envVars map[string]string
}

func (p *testOsProvider) Clock(context.Context) float64                          { return 0 }
func (p *testOsProvider) Time(context.Context, *vm.LuaTimeInput) (int64, *vm.LuaDateTime, error) {
	return 0, nil, nil
}
func (p *testOsProvider) Date(context.Context, string, int64) (string, error) { return "", nil }
func (p *testOsProvider) DateTable(context.Context, int64, bool) *vm.LuaDateTime { return nil }
func (p *testOsProvider) SetLocale(context.Context, string, string) (string, bool) {
	return "C", true
}
func (p *testOsProvider) Capabilities(context.Context) vm.LuaOsCaps {
	return vm.LuaOsCaps{AllowGetenv: true}
}
func (p *testOsProvider) Getenv(_ context.Context, name string) (string, bool) {
	v, ok := p.envVars[name]
	return v, ok
}

func TestPackagePath_VersionedEnvTakesPrecedence(t *testing.T) {
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_PATH":     "/generic/?.lua",
			"LUA_PATH_5_5": "/versioned/?.lua",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "/versioned/?.lua",
	"expected versioned path, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackagePath_FallsBackToGenericEnv(t *testing.T) {
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_PATH": "/generic/?.lua",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "/generic/?.lua",
	"expected generic path, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackagePath_DefaultWhenNoEnv(t *testing.T) {
	provider := &testOsProvider{
		envVars: map[string]string{},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "?.lua;?/init.lua",
	"expected default path, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackagePath_NoOsProvider(t *testing.T) {
	m := vm.New()
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "?.lua;?/init.lua",
	"expected default path without OS provider, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackageCpath_VersionedEnvTakesPrecedence(t *testing.T) {
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_CPATH":     "/generic/?.so",
			"LUA_CPATH_5_5": "/versioned/?.so",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.cpath == "/versioned/?.so",
	"expected versioned cpath, got: " .. package.cpath)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackageCpath_FallsBackToGenericEnv(t *testing.T) {
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_CPATH": "/generic/?.so",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.cpath == "/generic/?.so",
	"expected generic cpath, got: " .. package.cpath)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackagePath_EmptyVersionedOverridesGeneric(t *testing.T) {
	// Empty string is a valid value — should NOT fall through to generic
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_PATH_5_5": "",
			"LUA_PATH":     "/should_not_see/?.lua",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "",
	"expected empty path, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackagePath_DoubleQuestion_ReplacedWithDefault(t *testing.T) {
	// In Lua, ";;" in the env var is replaced with the default path
	provider := &testOsProvider{
		envVars: map[string]string{
			"LUA_PATH": "/custom/?.lua;;",
		},
	}
	m := vm.New()
	m.SetOsProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.path == "/custom/?.lua;?.lua;?/init.lua",
	"expected ;; expanded, got: " .. package.path)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
