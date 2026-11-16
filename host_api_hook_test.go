package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func hostRun(t *testing.T, v *vm.VM, name, src string) ([]vm.Value, error) {
	t.Helper()
	block, err := parser.Parse(name, src)
	if err != nil {
		t.Fatal(err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatal(err)
	}
	return v.Run(proto)
}

// Like lua_gettable or lua_compare called from C, a host API call fires no
// call or return hook of its own; only a metamethod it runs does. A hook that
// raises an error therefore cannot fail a lookup that runs no function.
func TestHostAPIFiresNoHookOfItsOwn(t *testing.T) {
	v := vm.New()
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	stdlib.Open(v)
	if _, err := hostRun(t, v, "=setup", `
		n = 0
		proxy = setmetatable({}, {__index = {a = 1}})
		fproxy = setmetatable({}, {__index = function() return 2 end})
		p1, p2 = {}, {}
		debug.sethook(function() n = n + 1 end, "cr")`); err != nil {
		t.Fatal(err)
	}
	proxy, fproxy := v.GetGlobal("proxy"), v.GetGlobal("fproxy")
	p1, p2 := v.GetGlobal("p1"), v.GetGlobal("p2")

	// The setup and count chunks account for 4 events between them (the
	// return from the first sethook, the call into and return out of the count
	// chunk, and the call to the second sethook); the host calls add none.
	if got, err := v.IndexValue(proxy, vm.NewString("a")); err != nil || got.AsInt() != 1 {
		t.Fatalf("IndexValue(proxy) = %v, %v", got, err)
	}
	if eq, err := v.CompareEQ(p1, p2); err != nil || eq {
		t.Fatalf("CompareEQ(p1, p2) = %v, %v", eq, err)
	}
	if err := v.SetIndexValue(vm.NewString("s"), vm.NewString("k"), vm.True); err == nil {
		t.Fatal("SetIndexValue on a string: expected an error")
	}
	res, err := hostRun(t, v, "=count", `debug.sethook() return n`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res[0].AsInt(); got != 4 {
		t.Fatalf("hook events = %d, want 4 (host calls must fire none)", got)
	}

	// A function metamethod is a real call and is hooked.
	if _, err := hostRun(t, v, "=setup2", `n = 0 debug.sethook(function() n = n + 1 end, "c")`); err != nil {
		t.Fatal(err)
	}
	if got, err := v.IndexValue(fproxy, vm.NewString("a")); err != nil || got.AsInt() != 2 {
		t.Fatalf("IndexValue(fproxy) = %v, %v", got, err)
	}
	res, err = hostRun(t, v, "=count2", `debug.sethook() return n`)
	if err != nil {
		t.Fatal(err)
	}
	// 1 for the __index call, 2 for the count chunk and its sethook call.
	if got := res[0].AsInt(); got != 3 {
		t.Fatalf("hook events with a function __index = %d, want 3", got)
	}

	// A hook that errors does not break a lookup that calls no function.
	if _, err := hostRun(t, v, "=errhook", `debug.sethook(function() error("hookerr") end, "c")`); err != nil {
		t.Fatal(err)
	}
	got, err := v.IndexValue(proxy, vm.NewString("a"))
	if err != nil || got.AsInt() != 1 {
		t.Fatalf("IndexValue(proxy) under an erroring hook = %v, %v", got, err)
	}
	if v.StackDepth() != 0 || v.Top() != 0 {
		t.Fatalf("left depth %d, top %d", v.StackDepth(), v.Top())
	}
}

// An error a host API call returns must not leave its call-stack snapshot
// behind for the next Run error's traceback.
func TestHostAPIErrorLeavesNoStaleTraceback(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	if _, err := hostRun(t, v, "=setup", `
		bad = setmetatable({}, {__index = function() error("from_index_mm") end})`); err != nil {
		t.Fatal(err)
	}
	if _, err := v.IndexValue(v.GetGlobal("bad"), vm.NewString("k")); err == nil {
		t.Fatal("expected the __index error")
	}
	if v.HasLastErrorTraceback() {
		t.Fatal("host API error left a call-stack snapshot")
	}
	if _, err := hostRun(t, v, "=later", `local function later_fn() error("later") end later_fn()`); err == nil {
		t.Fatal("expected the later error")
	}
	tb := v.TracebackFromLastError("", 0)
	if !strings.Contains(tb, "later_fn") {
		t.Fatalf("traceback does not show the later error's frames:\n%s", tb)
	}
}
