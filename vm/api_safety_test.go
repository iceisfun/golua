package vm

import (
	"context"
	"math"
	"testing"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
)

// These tests verify that exported Go API methods never panic.
// They exercise the error-returning boundaries added for Go consumers
// who call Table/Channel methods outside any ProtectedCall.

func TestTableSetNilKeyReturnsError(t *testing.T) {
	tbl := NewEmptyTable()
	err := tbl.Set(Nil, NewInt(1))
	if err == nil {
		t.Fatal("expected error for nil key")
	}
	if err.Error() != "table index is nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTableSetNaNKeyReturnsError(t *testing.T) {
	tbl := NewEmptyTable()
	err := tbl.Set(NewFloat(math.NaN()), NewInt(1))
	if err == nil {
		t.Fatal("expected error for NaN key")
	}
	if err.Error() != "table index is NaN" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTableSetValidKeyNoError(t *testing.T) {
	tbl := NewEmptyTable()
	if err := tbl.Set(NewString("key"), NewInt(42)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := tbl.Set(NewInt(1), NewString("val")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := tbl.Set(NewFloat(3.14), NewString("pi")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := tbl.Set(True, NewString("yes")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTableNextInvalidKeyReturnsError(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	_, _, err := tbl.Next(NewString("nonexistent"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if err.Error() != "invalid key to 'next'" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTableNextValidTraversalNoError(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	tbl.SetString("b", NewInt(2))

	k, v, err := tbl.Next(Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.IsNil() {
		t.Fatal("expected non-nil first key")
	}
	if v.IsNil() {
		t.Fatal("expected non-nil first value")
	}

	k2, _, err := tbl.Next(k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k2.IsNil() {
		t.Fatal("expected non-nil second key")
	}

	k3, _, err := tbl.Next(k2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !k3.IsNil() {
		t.Error("expected nil after last key")
	}
}

func TestTableDeleteNilKeyReturnsError(t *testing.T) {
	tbl := NewEmptyTable()
	err := tbl.Delete(Nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}

func TestChannelDoubleCloseReturnsError(t *testing.T) {
	provider := NewDefaultChanProvider()
	ch := provider.NewChannel(context.Background(), 0)
	if err := ch.Close(); err != nil {
		t.Fatalf("first close should succeed: %v", err)
	}
	err := ch.Close()
	if err == nil {
		t.Fatal("expected error on double close")
	}
	if err.Error() != "close of closed channel" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCatchesAllInternalPanics(t *testing.T) {
	// Verify that VM.Run never lets panics escape, even with code that
	// triggers internal errors (stack overflow, invalid operations).
	tests := []struct {
		name   string
		source string
	}{
		{"stack overflow", "local function f() f() end f()"},
		{"nil index", "local t = {} t[nil] = 1"},
		{"nan index", "local t = {} t[0/0] = 1"},
		{"call nil", "local f = nil f()"},
		{"index nil", "local x = nil return x.y"},
		{"arithmetic on string", "return 'hello' + 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic escaped VM.Run: %v", r)
				}
			}()
			block, err := parser.Parse("<test>", tt.source)
			if err != nil {
				t.Skipf("parse error (expected for some tests): %v", err)
			}
			proto, err := compiler.Compile("<test>", block)
			if err != nil {
				t.Skipf("compile error (expected for some tests): %v", err)
			}
			v := New(WithLimits(Limits{
				MaxCallDepth:  200,
				MaxStackSlots: 5000,
			}))
			_, err = v.Run(proto)
			if err == nil {
				t.Error("expected an error from this code")
			}
		})
	}
}

func TestProtectedCallCatchesAllPanics(t *testing.T) {
	tests := []struct {
		name string
		fn   Value
		args []Value
	}{
		{"nil value", Nil, nil},
		{"int value", NewInt(42), nil},
		{"string value", NewString("hello"), nil},
		{"bool value", True, nil},
		{"panicking native", NewNativeFunc(func(v *VM) int {
			panic("intentional panic")
		}), nil},
		{"error-raising native", NewNativeFunc(func(v *VM) int {
			panic(&LuaError{Value: NewString("lua error")})
		}), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic escaped ProtectedCall: %v", r)
				}
			}()
			v := New()
			_, err := v.ProtectedCall(tt.fn, tt.args)
			if err == nil {
				t.Error("expected an error")
			}
		})
	}
}
