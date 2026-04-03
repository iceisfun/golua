package stdlib

import (
	"context"
	"fmt"
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// testCodeProvider resolves chunks from a fixed map.
type testCodeProvider struct {
	files map[string]string
	caps  vm.LuaLoaderCaps
}

func (p *testCodeProvider) LoadChunk(_ context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
	if src, ok := p.files[name]; ok {
		return []byte(src), "@" + name, nil
	}
	return nil, "", fmt.Errorf("not found: %s", name)
}

func (p *testCodeProvider) Capabilities(_ context.Context) vm.LuaLoaderCaps {
	return p.caps
}

func TestSearchPath_WithCodeProvider(t *testing.T) {
	provider := &testCodeProvider{
		files: map[string]string{
			"mods/foo.lua": "return 42",
		},
	}
	m := vm.New()
	m.SetCodeProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local path = package.searchpath("foo", "mods/?.lua")
assert(path == "mods/foo.lua", "expected mods/foo.lua, got: " .. tostring(path))
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchPath_WithCodeProvider_NotFound(t *testing.T) {
	provider := &testCodeProvider{
		files: map[string]string{
			"mods/foo.lua": "return 42",
		},
	}
	m := vm.New()
	m.SetCodeProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local path, msg = package.searchpath("bar", "mods/?.lua;lib/?.lua")
assert(path == nil, "expected nil for missing module")
assert(type(msg) == "string")
assert(string.find(msg, "mods/bar.lua", 1, true))
assert(string.find(msg, "lib/bar.lua", 1, true))
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchPath_WithoutCodeProvider(t *testing.T) {
	m := vm.New()
	Open(m)

	err := runLuaWithVM(t, m, `
local path, msg = package.searchpath("foo", "?.lua")
assert(path == nil, "expected nil without provider")
assert(type(msg) == "string")
assert(string.find(msg, "foo.lua", 1, true))
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchPath_SepReplacement(t *testing.T) {
	provider := &testCodeProvider{
		files: map[string]string{
			"mods/foo/bar.lua": "return 1",
		},
	}
	m := vm.New()
	m.SetCodeProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local path = package.searchpath("foo.bar", "mods/?.lua")
assert(path == "mods/foo/bar.lua", "expected mods/foo/bar.lua, got: " .. tostring(path))
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
