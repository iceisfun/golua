package golua_test

// Go-side call boundary: what a native function sees on its own stack frame.

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// A native's optional trailing parameters must read nil, not whatever an
// earlier call left in those stack slots. Reference Lua's index2value returns
// the shared nil for any positive index at or past the frame's top.
func TestNativeUnpassedArgumentsAreNil(t *testing.T) {
	v := vm.New()

	var seen []string
	// A first call with a long argument list, then a second call at the same
	// base with one argument: the second must not see the first's tail.
	wide := vm.NewNativeFunc(func(*vm.VM) int { return 0 })
	narrow := vm.NewNativeFunc(func(v *vm.VM) int {
		for i := 1; i <= 8; i++ {
			seen = append(seen, v.Get(i).String())
		}
		return 0
	})

	if _, err := v.ProtectedCall(wide, []vm.Value{
		vm.NewString("a"), vm.NewString("b"), vm.NewString("c"), vm.NewString("d"),
		vm.NewString("e"), vm.NewString("f"), vm.NewInt(30),
	}); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := v.ProtectedCall(narrow, []vm.Value{vm.NewString("x")}); err != nil {
		t.Fatalf("second call: %v", err)
	}

	want := []string{"x", "nil", "nil", "nil", "nil", "nil", "nil", "nil"}
	for i, w := range want {
		if seen[i] != w {
			t.Fatalf("native argument %d read a previous call's value: got %v, want %v",
				i+1, seen, want)
		}
	}
}

// The same guarantee, reached from Lua rather than from Go.
func TestNativeUnpassedArgumentsAreNilFromLua(t *testing.T) {
	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)
	v.SetGlobal("wide", vm.NewNativeFunc(func(*vm.VM) int { return 0 }))
	v.SetGlobal("narrow", vm.NewNativeFunc(func(v *vm.VM) int {
		var parts []string
		for i := 1; i <= 8; i++ {
			parts = append(parts, v.Get(i).String())
		}
		v.Set(0, vm.NewString(strings.Join(parts, ",")))
		return 1
	}))

	// Both calls go through the same local, so the second native frame lands on
	// the register the first one used.
	proto := compileLua(t, `
		local f = wide
		f("a", "b", "c", "d", "e", "f", 30)
		f = narrow
		print(f("x"))

		-- and again through a tail call, whose frame is placed at the bare
		-- stack top rather than at a caller register
		local function tail(g, ...) return g(...) end
		tail(wide, "a", "b", "c", "d", "e", "f", 30)
		print(tail(narrow, "x"))
	`, "=stalearg")
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}

	const want = "x,nil,nil,nil,nil,nil,nil,nil"
	for _, line := range v.OutputLines() {
		if line != want {
			t.Fatalf("native argument slots retained a previous call's values:\n got: %q\nwant: %q",
				line, want)
		}
	}
}
