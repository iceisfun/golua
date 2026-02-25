-- test_tostring_special_float: tostring/print-style number rendering for inf/nan

assert(tostring(1/0) == "inf", "tostring(+inf) should be 'inf'")
assert(tostring(-1/0) == "-inf", "tostring(-inf) should be '-inf'")

local a = 0/0
local b = -a
assert(tostring(a) == "-nan", "tostring(0/0) should be '-nan'")
assert(tostring(b) == "nan", "tostring(-(0/0)) should be 'nan'")

-- %s uses tostring-style conversion in this implementation
assert(string.format("%s", 1/0) == "inf", "string.format('%s', +inf) should be 'inf'")
assert(string.format("%s", a) == "-nan", "string.format('%s', nan) should preserve sign")
