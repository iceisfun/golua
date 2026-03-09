package stdlib

import (
	"errors"
	"testing"

	"github.com/iceisfun/golua/vm"
)

type testLoadLibProvider struct {
	err      error
	callPath string
	callInit string
	caller   *vm.LuaCallerContext
	calls    int
}

func (p *testLoadLibProvider) LoadLib(path, init string, caller *vm.LuaCallerContext) (vm.NativeFunc, error) {
	p.calls++
	p.callPath = path
	p.callInit = init
	p.caller = caller
	if p.err != nil {
		return nil, p.err
	}
	return func(v *vm.VM) int {
		v.Set(0, vm.NewString("loaded:"+path+":"+init))
		return 1
	}, nil
}

func runLuaWithVM(t *testing.T, machine *vm.VM, source string) error {
	t.Helper()
	proto, err := compileTestLua(source)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	_, err = machine.Run(proto)
	return err
}

func TestPackageLoadlib_DefaultNil(t *testing.T) {
	m := vm.New()
	Open(m)

	err := runLuaWithVM(t, m, `
assert(package.loadlib == nil, "package.loadlib should be nil without provider")
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackageLoadlib_UsesProvider(t *testing.T) {
	provider := &testLoadLibProvider{}
	m := vm.New()
	m.SetLoadLibProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
assert(type(package.loadlib) == "function")
local loader, msg = package.loadlib("mod.so", "luaopen_mod")
assert(type(loader) == "function", "expected function loader")
assert(msg == nil, "expected nil error message")
assert(loader() == "loaded:mod.so:luaopen_mod")
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.calls)
	}
	if provider.callPath != "mod.so" || provider.callInit != "luaopen_mod" {
		t.Fatalf("unexpected provider args: path=%q init=%q", provider.callPath, provider.callInit)
	}
	if provider.caller == nil {
		t.Fatal("expected non-nil caller context")
	}
}

func TestPackageLoadlib_ProviderError(t *testing.T) {
	provider := &testLoadLibProvider{err: errors.New("blocked by policy")}
	m := vm.New()
	m.SetLoadLibProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local loader, msg = package.loadlib("x.so", "luaopen_x")
assert(loader == nil)
assert(type(msg) == "string")
assert(string.find(msg, "blocked by policy", 1, true) ~= nil)
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.calls)
	}
}

func TestPackageLoadlib_ArgValidation(t *testing.T) {
	provider := &testLoadLibProvider{}
	m := vm.New()
	m.SetLoadLibProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local ok, msg = pcall(package.loadlib)
assert(not ok and string.find(msg, "bad argument #1", 1, true))

ok, msg = pcall(package.loadlib, "a")
assert(not ok and string.find(msg, "bad argument #2", 1, true))

ok, msg = pcall(package.loadlib, 1, "luaopen_x")
assert(not ok and string.find(msg, "bad argument #1", 1, true))

ok, msg = pcall(package.loadlib, "x.so", {})
assert(not ok and string.find(msg, "bad argument #2", 1, true))
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider should not be called on bad args, got %d calls", provider.calls)
	}
}

func TestSetLoadLibProviderAccessors(t *testing.T) {
	m := vm.New()
	if m.LoadLibProvider() != nil {
		t.Fatal("expected nil provider by default")
	}

	provider := &testLoadLibProvider{}
	m.SetLoadLibProvider(provider)
	if m.LoadLibProvider() != provider {
		t.Fatal("LoadLibProvider getter did not return the configured provider")
	}
}

func TestPackageLoadlib_ErrorStringShape(t *testing.T) {
	provider := &testLoadLibProvider{err: errors.New("cannot dlopen /x.so")}
	m := vm.New()
	m.SetLoadLibProvider(provider)
	Open(m)

	err := runLuaWithVM(t, m, `
local loader, msg = package.loadlib("/x.so", "luaopen_x")
assert(loader == nil)
assert(msg == "cannot dlopen /x.so")
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
