package golua_test

// Regression tests for the Go embedding API: what a host sees from
// ProtectedCall/Run, how a host's own protected calls interact with a Lua
// xpcall in progress, custom LuaTable implementations, and the Lua number
// formatting of values converted to Go strings.

import (
	"math"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// Results handed to the host belong to the host: a later call into the VM, or
// a collection that clears VM-internal buffers, must not change them.
func TestGoAPIResultsSurviveLaterCalls(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	embedRun(t, v, `function f(x) return x, x + 1 end`)
	f := v.GetGlobal("f")

	r1, err := v.ProtectedCall(f, []vm.Value{vm.NewInt(1)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.ProtectedCall(f, []vm.Value{vm.NewInt(100)}); err != nil {
		t.Fatal(err)
	}
	if len(r1) != 2 || r1[0].AsInt() != 1 || r1[1].AsInt() != 2 {
		t.Fatalf("ProtectedCall results changed by a later call: %v", r1)
	}

	ra := embedRun(t, v, `return "a", "b"`)
	embedRun(t, v, `return "c", "d"`)
	if len(ra) != 2 || ra[0].AsString() != "a" || ra[1].AsString() != "b" {
		t.Fatalf("Run results changed by a later Run: %v", ra)
	}

	r1v := embedRun(t, v, `return 7`)
	embedRun(t, v, `return 8`)
	if len(r1v) != 1 || r1v[0].AsInt() != 7 {
		t.Fatalf("single Run result changed by a later Run: %v", r1v)
	}

	rt := embedRun(t, v, `return {}`)
	embedRun(t, v, `collectgarbage()`)
	if len(rt) != 1 || !rt[0].IsTable() {
		t.Fatalf("Run result wiped by collectgarbage: %v", rt)
	}
}

// A host native that calls ProtectedCall and handles the error itself is a
// lua_pcall with no message handler: a Lua xpcall around it must neither run
// its handler on that error nor lose its own error to it.
func TestGoAPIProtectedCallIgnoresOuterXpcallHandler(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	v.SetGlobal("try", vm.NewNativeFunc(func(v *vm.VM) int {
		_, err := v.ProtectedCall(v.Get(1), nil)
		if err != nil {
			v.Set(0, vm.NewString("caught: "+err.Error()))
		} else {
			v.Set(0, vm.NewString("ok"))
		}
		return 1
	}))

	res := embedRun(t, v, `
		local hits = 0
		local function h(m) hits = hits + 1 return "H(" .. tostring(m) .. ")" end
		local okA, eA = xpcall(function()
			return try(function() error("inner", 0) end)
		end, h)
		local hitsA = hits
		hits = 0
		local okB, eB = xpcall(function()
			local r = try(function() error("inner", 0) end)
			error("outer: " .. r, 0)
		end, h)
		return okA, eA, hitsA, okB, eB, hits`)
	got := make([]string, len(res))
	for i, r := range res {
		got[i] = r.String()
	}
	want := []string{"true", "caught: inner", "0", "false", "H(outer: caught: inner)", "1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// customTable is a LuaTable that is not a *vm.Table.
type customTable struct{ *vm.Table }

// The LuaTable interface is documented as open to custom implementations;
// Lua code must be able to index, assign to, and use the table library on
// one, as it can on any table.
func TestGoAPICustomLuaTableIndexing(t *testing.T) {
	v := vm.New()
	stdlib.Open(v)
	ct := customTable{vm.NewEmptyTable()}
	if err := ct.Set(vm.NewString("k"), vm.NewString("v")); err != nil {
		t.Fatal(err)
	}
	v.SetGlobal("t", vm.NewTable(ct))

	cases := []struct{ src, want string }{
		{`return t.k`, "v"},
		{`local key = "k" return t[key]`, "v"},
		{`t.x = 1 return rawget(t, "x")`, "1"},
		{`local key = "y" t[key] = 2 return rawget(t, "y")`, "2"},
		{`t[1] = "one" return t[1]`, "one"},
		{`table.insert(t, "two") return t[2]`, "two"},
		{`table.remove(t) return #t`, "1"},
		{`local _ENV = t return k`, "v"},
		{`function t.m(self) return self.k end return t:m()`, "v"},
		{`local ct = t setmetatable(ct, {__index = function(_, key) return "mm:" .. key end}) return ct.missing`, "mm:missing"},
	}
	for _, tc := range cases {
		res, err := v.Run(embedProto(t, tc.src))
		if err != nil {
			t.Fatalf("%s: %v", tc.src, err)
		}
		if len(res) != 1 || res[0].String() != tc.want {
			t.Fatalf("%s: got %v, want %s", tc.src, res, tc.want)
		}
	}
}

// ValueToString renders numbers the way Lua's tostring does, since its
// result reaches Lua code (require names, searcher messages, load errors)
// and hosts (LuaError.Error).
func TestGoAPIValueToStringFloats(t *testing.T) {
	cases := []struct {
		f    float64
		want string
	}{
		{2.0, "2.0"},
		{-0.5, "-0.5"},
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
		{0x1p63, "9.2233720368547758e+18"},
		{0.1, "0.1"},
		{1e100, "1e+100"},
	}
	for _, tc := range cases {
		if got := vm.ValueToString(vm.NewFloat(tc.f)); got != tc.want {
			t.Errorf("ValueToString(%v) = %q, want %q", tc.f, got, tc.want)
		}
	}
	if got := (&vm.LuaError{Value: vm.NewFloat(2)}).Error(); got != "2.0" {
		t.Errorf("LuaError{2.0}.Error() = %q, want %q", got, "2.0")
	}

	v := vm.New()
	stdlib.Open(v)
	res := embedRun(t, v, `
		package.preload["2.0"] = function(name) return "loaded " .. name end
		return require(2.0)`)
	if len(res) < 1 || res[0].String() != "loaded 2.0" {
		t.Fatalf("require(2.0): got %v, want loaded 2.0", res)
	}
}
