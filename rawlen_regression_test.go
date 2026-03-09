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
