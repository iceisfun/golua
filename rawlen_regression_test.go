package golua_test

import "testing"

func TestRawlenRegression_ThreadRaisesTypeError(t *testing.T) {
	source := `
		local co = coroutine.create(function() end)
		local ok, err = pcall(rawlen, co)
		assert(ok == false)
		assert(tostring(err):find("table or string expected, got thread", 1, true), tostring(err))
	`
	runLuaSource(t, source, "test_rawlen_thread_type_error")
}

func TestRawAccessRegression_ThreadRaisesTypeError(t *testing.T) {
	source := `
		local co = coroutine.create(function() end)
		local ok1, err1 = pcall(rawget, co, "x")
		local ok2, err2 = pcall(rawset, co, "x", 1)
		assert(ok1 == false)
		assert(ok2 == false)
		assert(tostring(err1):find("table expected, got thread", 1, true), tostring(err1))
		assert(tostring(err2):find("table expected, got thread", 1, true), tostring(err2))
	`
	runLuaSource(t, source, "test_raw_access_thread_type_error")
}

func TestSetMetatableRegression_ThreadRaisesTypeError(t *testing.T) {
	source := `
		local co = coroutine.create(function() end)
		local ok, err = pcall(setmetatable, co, {})
		assert(ok == false)
		assert(tostring(err):find("table expected, got thread", 1, true), tostring(err))
	`
	runLuaSource(t, source, "test_setmetatable_thread_type_error")
}
