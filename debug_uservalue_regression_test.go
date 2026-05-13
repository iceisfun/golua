package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

func TestDebugUservalueRegression_GetAndSetUservalue(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	block, err := parser.Parse("test_debug_uservalue_get_set", `
		assert(type(debug.getuservalue) == "function")
		assert(type(debug.setuservalue) == "function")

		-- getuservalue on non-userdata returns nil (1 value)
		assert(debug.getuservalue({}) == nil)

		-- setuservalue returns the userdata on success
		local ret = debug.setuservalue(ud, 123)
		assert(ret == ud, "setuservalue should return the userdata, got " .. tostring(ret))

		-- getuservalue returns value and true for valid slot
		local val, exists = debug.getuservalue(ud, 1)
		assert(val == 123, "expected 123, got " .. tostring(val))
		assert(exists == true, "expected true for valid slot")

		local ret2 = debug.setuservalue(ud, "abc")
		assert(ret2 == ud)
		local val2, exists2 = debug.getuservalue(ud)
		assert(val2 == "abc")
		assert(exists2 == true)

		-- setuservalue on non-userdata errors with "userdata expected" (matches reference Lua)
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

func TestDebugUservalueRegression_ZeroSlotUserdata(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	block, err := parser.Parse("test_zero_slot_ud", `
		-- ud0 has 0 user value slots
		local val = debug.getuservalue(ud0, 1)
		assert(val == nil, "expected nil for 0-slot ud, got " .. tostring(val))

		-- setuservalue on 0-slot userdata returns nil
		local ret = debug.setuservalue(ud0, 42)
		assert(ret == nil, "expected nil for 0-slot ud setuservalue, got " .. tostring(ret))

		local ret2 = debug.setuservalue(ud0, 42, 1)
		assert(ret2 == nil, "expected nil for n=1 on 0-slot ud")
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test_zero_slot_ud", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetDebugProvider(provider)
	stdlib.Open(v)
	v.SetGlobal("ud0", vm.NewUserdataValueUV(struct{}{}, nil, 0))
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}
