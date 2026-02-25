-- test_string_format_hex_float: %a/%A should format floats in hex form

assert(string.format("%a", 1.5) == "0x1.8p+0", "%a for 1.5 should be 0x1.8p+0")
assert(string.format("%A", 1.5) == "0X1.8P+0", "%A for 1.5 should be 0X1.8P+0")
assert(string.format("%a", -16.0) == "-0x1p+4", "%a for -16 should be -0x1p+4")

local ok, err = pcall(function() return string.format("%a", "nope") end)
assert(ok == false, "%a with non-number should error")
assert(type(err) == "string" and err:find("number expected"),
       "unexpected %a error: " .. tostring(err))
