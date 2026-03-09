package golua_test

import "testing"

func TestLoadReaderRegression_XPCallWrapKeepsRawReaderError(t *testing.T) {
	source := `
		local function reader()
			error("rboom")
		end
		local ok, f, err = xpcall(function()
			return load(reader)
		end, function(e) return e end)
		assert(ok == true)
		assert(f == nil)
		assert(type(err) == "string", type(err))
		assert(err:find(":3: rboom", 1, true), err)
		assert(err:find("stack traceback:", 1, true) == nil, err)
	`
	runLuaSource(t, source, "test_load_reader_xpcall_raw_error")
}
