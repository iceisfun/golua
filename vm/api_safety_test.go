package vm

import (
	"context"
	"math"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
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

// The table, length and comparison methods run metamethods. Called from host
// code with no call in progress, an error a metamethod raises must come back
// as the returned error, and the VM must be left as it was.
func TestHostAPIMetamethodErrorsReturned(t *testing.T) {
	v := New(WithLimits(Limits{MaxCallDepth: 200}))
	compile := func(src string) Value {
		block, err := parser.Parse("=host", src)
		if err != nil {
			t.Fatal(err)
		}
		proto, err := compiler.Compile("=host", block)
		if err != nil {
			t.Fatal(err)
		}
		res, err := v.Run(proto)
		if err != nil {
			t.Fatal(err)
		}
		return res[0]
	}
	luaErr := compile("return function() local x; return x.y end")
	recurse := compile("return function(t, k) return t[k] end")
	nativeErr := NewNativeFunc(func(*VM) int { panic("native err") })

	ops := map[string]func(obj Value) error{
		"IndexValue": func(obj Value) error { _, err := v.IndexValue(obj, NewString("k")); return err },
		"IndexInt":   func(obj Value) error { _, err := v.IndexInt(obj, 1); return err },
		"TableGet":   func(obj Value) error { _, err := v.TableGet(obj.AsTable(), NewString("k")); return err },
		"TableGetInt": func(obj Value) error {
			_, err := v.TableGetInt(obj.AsTable(), 1)
			return err
		},
		"SetIndexValue": func(obj Value) error { return v.SetIndexValue(obj, NewString("k"), True) },
		"SetIndexInt":   func(obj Value) error { return v.SetIndexInt(obj, 1, True) },
		"TableSetInt":   func(obj Value) error { return v.TableSetInt(obj.AsTable(), 1, True) },
		"ObjLen":        func(obj Value) error { _, err := v.ObjLen(obj); return err },
		"CompareEQ":     func(obj Value) error { _, err := v.CompareEQ(obj, NewTable(NewEmptyTable())); return err },
		"CompareLT":     func(obj Value) error { _, err := v.CompareLT(obj, obj); return err },
	}
	for name, op := range ops {
		for mmName, mm := range map[string]Value{"lua": luaErr, "native": nativeErr, "recursive": recurse} {
			mt := NewEmptyTable()
			for _, ev := range []string{MetaIndex, MetaNewIndex, MetaLen, MetaEq, MetaLt} {
				mt.SetString(ev, mm)
			}
			tbl := NewEmptyTable()
			tbl.SetMetatable(mt)
			obj := NewTable(tbl)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("%s with a %s metamethod: panic escaped: %v", name, mmName, r)
					}
				}()
				if err := op(obj); err == nil {
					t.Fatalf("%s with a %s metamethod: expected an error", name, mmName)
				}
			}()
			if v.StackDepth() != 0 || v.Top() != 0 {
				t.Fatalf("%s with a %s metamethod: left depth %d, top %d", name, mmName, v.StackDepth(), v.Top())
			}
		}
	}
	if got := compile("return 1 + 1"); got.AsInt() != 2 {
		t.Fatalf("VM unusable afterwards: %v", got)
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
