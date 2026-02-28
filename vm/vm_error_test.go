package vm

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

func TestRuntimeErrorCallNilIncludesSourceLocation(t *testing.T) {
	// Line 1: local x = nil
	// Line 2: -- blank
	// Line 3: x()
	src := "local x = nil\n\nx()\n"
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test.lua", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := New()
	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "test.lua:3:") {
		t.Errorf("expected error to contain 'test.lua:3:', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "attempt to call a nil value") {
		t.Errorf("expected error to contain 'attempt to call a nil value', got: %s", errMsg)
	}
}

func TestRuntimeErrorCallNonFunctionIncludesSourceLocation(t *testing.T) {
	src := "local x = 42\nx()\n"
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("myfile.lua", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := New()
	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "myfile.lua:2:") {
		t.Errorf("expected error to contain 'myfile.lua:2:', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "attempt to call a number value") {
		t.Errorf("expected error to contain 'attempt to call a number value', got: %s", errMsg)
	}
}

func TestRuntimeErrorCallNilViaPcall(t *testing.T) {
	// pcall should also preserve the source location in the error message
	src := `local ok, err = pcall(function()
	local x = nil
	x()
end)
return ok, err
`
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("pcall_test.lua", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := New()
	// pcall needs a basic setup - register pcall as native
	v.SetGlobal("pcall", NewNativeFunc(func(vm *VM) int {
		argc := vm.ArgCount()
		fn := vm.Get(1)
		args := make([]Value, argc-1)
		for i := 2; i <= argc; i++ {
			args[i-2] = vm.Get(i)
		}
		results, callErr := vm.ProtectedCall(fn, args)
		if callErr != nil {
			vm.Set(0, False)
			if le, ok := callErr.(*LuaError); ok {
				vm.Set(1, le.Value)
			} else {
				vm.Set(1, NewString(callErr.Error()))
			}
			return 2
		}
		vm.Set(0, True)
		for i, r := range results {
			vm.Set(i+1, r)
		}
		return 1 + len(results)
	}))

	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].AsBool() {
		t.Fatal("expected pcall to return false")
	}

	errMsg := results[1].AsString()
	if !strings.Contains(errMsg, "pcall_test.lua:3:") {
		t.Errorf("expected error to contain 'pcall_test.lua:3:', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "attempt to call a nil value") {
		t.Errorf("expected error to contain 'attempt to call a nil value', got: %s", errMsg)
	}
}
