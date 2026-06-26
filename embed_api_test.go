package golua_test

// Property/invariant tests for golua's *embedding* surface — the Go<->Lua API
// (value marshaling, native-function calling conventions, calling Lua from Go,
// and the sandbox guarantee that a panicking Go native is catchable from Lua).
// This is the surface a real embedder uses; it has no PUC-Lua equivalent, so it
// is tested by round-trip/property invariants rather than differentially.

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func embedProto(t *testing.T, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("=embed", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p, err := compiler.Compile("=embed", block)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return p
}

func embedRun(t *testing.T, v *vm.VM, src string) []vm.Value {
	t.Helper()
	res, err := v.Run(embedProto(t, src))
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	return res
}

// --- value marshaling round-trips: Go -> Lua global -> back through Lua -------
func TestEmbedMarshalRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		val  vm.Value
		// luaCheck returns a Lua expression that must evaluate true given global x
		check string
	}{
		{"nil", vm.Nil, "x == nil"},
		{"true", vm.NewBool(true), "x == true"},
		{"false", vm.NewBool(false), "x == false"},
		{"int0", vm.NewInt(0), "x == 0 and math.type(x) == 'integer'"},
		{"int_max", vm.NewInt(math.MaxInt64), "x == 0x7fffffffffffffff"},
		{"int_min", vm.NewInt(math.MinInt64), "x == math.mininteger"},
		{"int_neg", vm.NewInt(-1), "x == -1 and math.type(x) == 'integer'"},
		{"float", vm.NewFloat(3.5), "x == 3.5 and math.type(x) == 'float'"},
		{"float_int_valued", vm.NewFloat(2.0), "x == 2.0 and math.type(x) == 'float'"},
		{"inf", vm.NewFloat(math.Inf(1)), "x == math.huge"},
		{"neg_inf", vm.NewFloat(math.Inf(-1)), "x == -math.huge"},
		{"nan", vm.NewFloat(math.NaN()), "x ~= x"},
		{"str_empty", vm.NewString(""), "x == '' and #x == 0"},
		{"str_ascii", vm.NewString("hello"), "x == 'hello' and #x == 5"},
		{"str_embedded_nul", vm.NewString("a\x00b"), "#x == 3 and x:byte(2) == 0"},
		{"str_binary", vm.NewString("\x80\xff\x01\x7f"), "#x == 4 and x:byte(1) == 128 and x:byte(2) == 255"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := vm.New()
			stdlib.Open(v)
			v.SetGlobal("x", tc.val)
			res := embedRun(t, v, "return ("+tc.check+")")
			if len(res) != 1 || !res[0].AsBool() {
				t.Fatalf("round-trip %s: Lua check %q failed (got %v)", tc.name, tc.check, res)
			}
			// And the value read straight back from the global is identical.
			got := v.GetGlobal("x")
			if got.Type() != tc.val.Type() {
				t.Fatalf("%s: GetGlobal type %s != %s", tc.name, got.Type(), tc.val.Type())
			}
		})
	}
}

// --- Lua -> Go: values produced by Lua read back with correct Go type/value --
func TestEmbedReadbackTypes(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	res := embedRun(t, v, "return 42, 3.5, 'hi', true, nil, 'a\\0b'")
	if len(res) < 6 {
		t.Fatalf("want 6 results, got %d", len(res))
	}
	if !res[0].IsInt() || res[0].AsInt() != 42 {
		t.Errorf("int: %v", res[0])
	}
	if !res[1].IsFloat() || res[1].AsFloat() != 3.5 {
		t.Errorf("float: %v", res[1])
	}
	if !res[2].IsString() || res[2].AsString() != "hi" {
		t.Errorf("string: %v", res[2])
	}
	if !res[3].IsBool() || !res[3].AsBool() {
		t.Errorf("bool: %v", res[3])
	}
	if !res[4].IsNil() {
		t.Errorf("nil: %v", res[4])
	}
	if !res[5].IsString() || res[5].AsString() != "a\x00b" {
		t.Errorf("nul string: %q", res[5].AsString())
	}
}

// --- native function calling conventions ------------------------------------
func TestEmbedNativeConventions(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)

	// arg count + access + multi-return
	v.SetGlobal("sum3", vm.NewNativeFunc(func(v *vm.VM) int {
		if v.ArgCount() != 3 {
			v.Set(0, vm.NewString("bad argc"))
			return 1
		}
		s := v.Get(1).AsInt() + v.Get(2).AsInt() + v.Get(3).AsInt()
		v.Set(0, vm.NewInt(s))
		v.Set(1, vm.NewInt(v.Get(1).AsInt()))
		return 2
	}))
	res := embedRun(t, v, "local a, b = sum3(10, 20, 30); return a, b")
	if res[0].AsInt() != 60 || res[1].AsInt() != 10 {
		t.Fatalf("multi-return: %v", res)
	}

	// missing args read as nil
	v.SetGlobal("argkind", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewString(v.Get(1).Type()))
		v.Set(1, vm.NewString(v.Get(5).Type())) // never passed -> nil
		return 2
	}))
	res = embedRun(t, v, "return argkind('x')")
	if res[0].AsString() != "string" || res[1].AsString() != "nil" {
		t.Fatalf("missing-arg nil: %v", res)
	}

	// method-call convention: self is Get(1)
	v.SetGlobal("mkobj", vm.NewNativeFunc(func(v *vm.VM) int {
		mt := vm.NewEmptyTable()
		mt.SetString("val", func() vm.Value {
			return vm.NewNativeFunc(func(v *vm.VM) int {
				self := v.Get(1).AsTable() // self is first arg on a method call
				v.Set(0, self.Get(vm.NewString("n")))
				return 1
			})
		}())
		obj := vm.NewEmptyTable()
		obj.SetString("n", vm.NewInt(7))
		obj.SetString("val", mt.GetString("val"))
		v.Set(0, vm.NewTable(obj))
		return 1
	}))
	res = embedRun(t, v, "local o = mkobj(); return o:val()")
	if res[0].AsInt() != 7 {
		t.Fatalf("method self convention: %v", res)
	}
}

// --- THE sandbox guarantee at the API level: a panicking Go native must be a
// CATCHABLE Lua error under pcall, never an uncatchable host crash. -----------
func TestEmbedNativePanicIsCatchable(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)

	v.SetGlobal("boom_str", vm.NewNativeFunc(func(v *vm.VM) int {
		panic("boom-from-go")
	}))
	v.SetGlobal("boom_err", vm.NewNativeFunc(func(v *vm.VM) int {
		panic(errors.New("err-from-go"))
	}))
	v.SetGlobal("boom_nil", vm.NewNativeFunc(func(v *vm.VM) int {
		var p *int
		_ = *p // nil deref -> runtime panic
		return 0
	}))

	for _, fn := range []string{"boom_str", "boom_err", "boom_nil"} {
		res := embedRun(t, v, "local ok, err = pcall("+fn+"); return ok, tostring(err)")
		if res[0].AsBool() {
			t.Fatalf("%s: pcall returned true — a panicking Go native escaped catching", fn)
		}
		if res[1].AsString() == "" {
			t.Fatalf("%s: caught error has no message", fn)
		}
	}

	// The VM must remain usable after catching a Go-native panic.
	res := embedRun(t, v, "return 1 + 1")
	if res[0].AsInt() != 2 {
		t.Fatalf("VM unusable after caught native panic: %v", res)
	}
}

// --- calling Lua from Go via ProtectedCall ----------------------------------
func TestEmbedCallLuaFromGo(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	embedRun(t, v, "function adder(a, b) return a + b, a * b end")
	fn := v.GetGlobal("adder")
	if !fn.IsCallable() {
		t.Fatalf("adder not callable: %v", fn)
	}
	res, err := v.ProtectedCall(fn, []vm.Value{vm.NewInt(6), vm.NewInt(7)})
	if err != nil {
		t.Fatalf("ProtectedCall: %v", err)
	}
	if len(res) != 2 || res[0].AsInt() != 13 || res[1].AsInt() != 42 {
		t.Fatalf("call-Lua-from-Go: %v", res)
	}

	// A Lua error in the called function is returned, not panicked.
	embedRun(t, v, "function boom() error('lua-boom') end")
	_, err = v.ProtectedCall(v.GetGlobal("boom"), nil)
	if err == nil || !strings.Contains(err.Error(), "lua-boom") {
		t.Fatalf("expected caught lua-boom error, got %v", err)
	}
}

// --- provider gating: nil provider => module absent (sandbox default) --------
func TestEmbedProviderGating(t *testing.T) {
	v := vm.New() // no providers set
	stdlib.Open(v)
	// os/io are gated behind providers; with none, the capability is absent.
	res := embedRun(t, v, "return type(os) == 'nil' or type(os.execute) == 'nil'")
	if !res[0].AsBool() {
		t.Fatalf("expected os/os.execute absent without a provider")
	}
}
