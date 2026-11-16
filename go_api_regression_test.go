package golua_test

// Regression tests for the Go embedding API: what a host sees from
// ProtectedCall/Run, how a host's own protected calls interact with a Lua
// xpcall in progress, custom LuaTable implementations, and the Lua number
// formatting of values converted to Go strings.

import (
	"math"
	"testing"

	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

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
