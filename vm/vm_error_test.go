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

// expectRuntimeError compiles+runs src and checks the error contains both wantLoc and wantMsg.
func expectRuntimeError(t *testing.T, filename, src, wantLoc, wantMsg string) {
	t.Helper()
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(filename, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := New()
	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, wantLoc) {
		t.Errorf("expected error to contain %q, got: %s", wantLoc, errMsg)
	}
	if !strings.Contains(errMsg, wantMsg) {
		t.Errorf("expected error to contain %q, got: %s", wantMsg, errMsg)
	}
}

func TestRuntimeErrorIndexNil(t *testing.T) {
	expectRuntimeError(t, "idx.lua",
		"local x = nil\n\nlocal _ = x.foo\n",
		"idx.lua:3:", "attempt to index a nil value")
}

func TestRuntimeErrorIndexNumber(t *testing.T) {
	expectRuntimeError(t, "idx2.lua",
		"local x = 42\nlocal _ = x.foo\n",
		"idx2.lua:2:", "attempt to index a number value")
}

func TestRuntimeErrorIndexSetNil(t *testing.T) {
	expectRuntimeError(t, "idxset.lua",
		"local x = nil\n\nx.foo = 1\n",
		"idxset.lua:3:", "attempt to index a nil value")
}

func TestRuntimeErrorArithmeticOnNil(t *testing.T) {
	expectRuntimeError(t, "arith.lua",
		"local x = nil\nlocal y = x + 1\n",
		"arith.lua:2:", "attempt to perform arithmetic on a nil value")
}

func TestRuntimeErrorArithmeticOnString(t *testing.T) {
	expectRuntimeError(t, "arith2.lua",
		"local x = \"hello\"\nlocal y = x + 1\n",
		"arith2.lua:2:", "attempt to perform arithmetic on a string value")
}

func TestRuntimeErrorArithmeticUnaryOnNil(t *testing.T) {
	expectRuntimeError(t, "unm.lua",
		"local x = nil\nlocal y = -x\n",
		"unm.lua:2:", "attempt to perform arithmetic on a nil value")
}

func TestRuntimeErrorBitwiseOnNil(t *testing.T) {
	expectRuntimeError(t, "bit.lua",
		"local x = nil\nlocal y = x & 1\n",
		"bit.lua:2:", "attempt to perform bitwise operation on a nil value")
}

func TestRuntimeErrorBitwiseNotOnNil(t *testing.T) {
	expectRuntimeError(t, "bnot.lua",
		"local x = nil\nlocal y = ~x\n",
		"bnot.lua:2:", "attempt to perform bitwise operation on a nil value")
}

func TestRuntimeErrorLengthOfNil(t *testing.T) {
	expectRuntimeError(t, "len.lua",
		"local x = nil\nlocal y = #x\n",
		"len.lua:2:", "attempt to get length of a nil value")
}

func TestRuntimeErrorCompareNilWithNumber(t *testing.T) {
	expectRuntimeError(t, "cmp.lua",
		"local x = nil\nif x < 1 then end\n",
		"cmp.lua:2:", "attempt to compare")
}

func TestRuntimeErrorConcatenateNil(t *testing.T) {
	expectRuntimeError(t, "cat.lua",
		"local x = nil\nlocal y = x .. \"hi\"\n",
		"cat.lua:2:", "attempt to concatenate a nil value")
}

func TestRuntimeErrorIntegerFloorDivByZero(t *testing.T) {
	expectRuntimeError(t, "idiv.lua",
		"local x = 10\nlocal y = x // 0\n",
		"idiv.lua:2:", "attempt to divide by zero")
}

func TestRuntimeErrorIntegerModByZero(t *testing.T) {
	expectRuntimeError(t, "mod.lua",
		"local x = 10\nlocal y = x % 0\n",
		"mod.lua:2:", "attempt to perform 'n%0'")
}
