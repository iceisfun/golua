-- test_string_format_special_float: inf/nan formatting and invalid %F

assert(string.format("%f", 1/0) == "inf", "%f +inf should be 'inf'")
assert(string.format("%+f", 1/0) == "+inf", "%+f +inf should be '+inf'")
assert(string.format("%8f", 1/0) == "     inf", "%8f +inf width mismatch")
assert(string.format("%-8f", 1/0) == "inf     ", "%-8f +inf width mismatch")
assert(string.format("%f", -1/0) == "-inf", "%f -inf should be '-inf'")
assert(string.format("%f", 0/0) == "-nan", "%f nan should be '-nan'")

assert(string.format("%E", 1/0) == "INF", "%E +inf should be 'INF'")
assert(string.format("%G", -1/0) == "-INF", "%G -inf should be '-INF'")
assert(string.format("%A", 0/0) == "-NAN", "%A nan should be '-NAN'")

local ok, err = pcall(function() return string.format("%F", 1.0) end)
assert(ok == false, "%F should be invalid in Lua 5.4")
assert(type(err) == "string" and err:find("invalid conversion") and err:find("%%F"),
       "unexpected %F error: " .. tostring(err))
