-- Bug 1: %q with floats uses decimal instead of hex float format.
-- Lua 5.4 uses %a (hex float) for exact binary roundtrip.
-- Critical sub-bug: %q with 1.0 produces "1" which load()s as integer, not float.

-- %q of a float must roundtrip as a float, not an integer
local s = string.format("%q", 1.0)
local f = load("return " .. s)
assert(f, "load of %q float should work: " .. s)
local val = f()
assert(math.type(val) == "float",
  "%q of 1.0 should roundtrip as float, got " .. math.type(val) .. " from: " .. s)
assert(val == 1.0, "%q of 1.0 should roundtrip to 1.0")

-- %q of pi should roundtrip exactly
local s2 = string.format("%q", math.pi)
local f2 = load("return " .. s2)
assert(f2, "load of %q pi should work: " .. s2)
assert(f2() == math.pi, "%q of pi should roundtrip exactly")

-- Bug 2: %q with math.mininteger uses decimal instead of hex.
-- "-9223372036854775808" can't be parsed as int literal (positive part overflows)
local s3 = string.format("%q", math.mininteger)
local f3 = load("return " .. s3)
assert(f3, "load of %q mininteger should work: " .. s3)
local val3 = f3()
assert(val3 == math.mininteger,
  "%q of mininteger should roundtrip, got: " .. tostring(val3))
assert(math.type(val3) == "integer",
  "%q of mininteger should roundtrip as integer, got " .. math.type(val3))

-- Bug 3: %q accepts tables and functions instead of erroring
local ok, err = pcall(string.format, "%q", {})
assert(not ok, "%q of table should error")
assert(tostring(err):find("no literal form") or tostring(err):find("has no literal"),
  "%q table error should mention 'no literal form', got: " .. tostring(err))

local ok2, err2 = pcall(string.format, "%q", print)
assert(not ok2, "%q of function should error")

print("PASS")
