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

// Reference lua_load runs the reader under the active message handler, so
// the error load returns from inside an xpcall is the handler's result, and
// the handler stays armed for the xpcall's own error.
func TestLoadReaderRegression_XPCallHandlerShapesReaderError(t *testing.T) {
	source := `
		local hits = 0
		local function h(m) hits = hits + 1 return "H(" .. tostring(m) .. ")" end
		local ok, f, err = xpcall(function()
			return load(function() error("rd", 0) end)
		end, h)
		assert(ok == true and f == nil, tostring(f))
		assert(err == "H(rd)", err)
		assert(hits == 1, hits)

		local marker = {}
		ok, f, err = xpcall(function()
			return load(function() error("rd", 0) end)
		end, function() return marker end)
		assert(ok == true and f == nil and err == marker, tostring(err))

		ok, err = xpcall(function()
			local _, e = load(function() error("rd", 0) end)
			error("after:" .. e, 0)
		end, function(m) return "H:" .. m end)
		assert(ok == false and err == "H:after:H:rd", err)
	`
	runLuaSource(t, source, "test_load_reader_xpcall_handler_result")
}

// A nil handler result becomes the reference "<no error object>" string, and
// load's own "reader function must return a string" error goes through the
// handler like an error the reader raises.
func TestLoadReaderRegression_XPCallHandlerNilResultAndTypeError(t *testing.T) {
	source := `
		local ok, f, err = xpcall(function()
			return load(function() error("rd", 0) end)
		end, function() return nil end)
		assert(ok == true and f == nil and err == "<no error object>", tostring(err))

		ok, f, err = xpcall(function()
			return load(function() return {} end)
		end, function(m) return "H(" .. m .. ")" end)
		assert(ok == true and f == nil, tostring(f))
		assert(err:match("^H%(.-reader function must return a string%)$"), err)

		-- a pcall in between keeps the handler out of it
		ok, f, err = xpcall(function()
			return pcall(load, function() return {} end)
		end, function(m) return "H(" .. m .. ")" end)
		assert(ok == true and f == true and err == nil, tostring(err))
	`
	runLuaSource(t, source, "test_load_reader_xpcall_nil_and_type_error")
}
