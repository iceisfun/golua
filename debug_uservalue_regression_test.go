package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func TestDebugUservalueRegression_GetAndSetUservalue(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	block, err := parser.Parse("test_debug_uservalue_get_set", `
		assert(type(debug.getuservalue) == "function")
		assert(type(debug.setuservalue) == "function")

		assert(debug.getuservalue({}) == nil)

		local old = debug.setuservalue(ud, 123)
		assert(old == nil, tostring(old))
		assert(debug.getuservalue(ud) == 123)

		local old2 = debug.setuservalue(ud, "abc")
		assert(old2 == 123, tostring(old2))
		assert(debug.getuservalue(ud) == "abc")

		local ok, err = pcall(debug.setuservalue, {}, 1)
		assert(ok == false)
		assert(tostring(err):find("userdata expected, got table", 1, true), tostring(err))
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test_debug_uservalue_get_set", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetDebugProvider(provider)
	stdlib.Open(v)
	v.SetGlobal("ud", vm.NewUserdataValue(struct{}{}, nil))
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}
