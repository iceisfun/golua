-- string.format missing/broken specifiers

-- %i should work like %d
assert(string.format("%i", -12) == "-12", "format %i")

-- %u unsigned integer
assert(string.format("-%u-", 55) == "-55-", "format %u")

-- %p should return (null) for non-reference types
assert(string.format("%p", 1) == "(null)", "format %p int")
assert(string.format("%p", true) == "(null)", "format %p bool")
assert(string.format("%p", nil) == "(null)", "format %p nil")

-- %p should return hex pointer for tables
local t = {}
local r = string.format("=%p=", t)
assert(r:match("^=0x[0-9a-f]+="), "format %p table should give hex pointer")

-- %q with nil should give unquoted nil
assert(string.format("[%q]", nil) == "[nil]", "format %q nil")

-- %q with false should give unquoted false
assert(string.format("%q", false) == "false", "format %q false")

-- %q with infinity
assert(string.format("%q,%q", math.huge, -math.huge) == "1e9999,-1e9999", "format %q inf")

-- %q with NaN
assert(string.format("%q", 0/0) == "(0/0)", "format %q nan")

-- Not enough values should error
assert(not pcall(string.format, "%s %d", 1), "format not enough values should error")
