package vm

import (
	"testing"
)

func TestVM_RetBufReentrancy(t *testing.T) {
	v := New()

	// Create a native function that calls another Lua function
	// and checks if its own local state is preserved.
	innerFunc := NewNativeFunc(func(v *VM) int {
		v.Set(0, NewString("inner"))
		return 1
	})

	outerFunc := NewNativeFunc(func(v *VM) int {
		// 1. Call inner (uses retBuf)
		v.ProtectedCall(innerFunc, nil)

		// 2. Return something from outer
		v.Set(0, NewString("outer"))
		return 1
	})

	v.SetGlobal("outer", outerFunc)

	// Running this via ProtectedCall initiates the first level of retBuf usage
	res, err := v.ProtectedCall(outerFunc, nil)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if len(res) == 0 || res[0].AsString() != "outer" {
		t.Errorf("Expected 'outer', got %v", res)
	}
}
