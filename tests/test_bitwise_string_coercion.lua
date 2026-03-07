-- Bug: Bitwise operations incorrectly coerce strings to integers.
-- Lua 5.4 specifies that bitwise ops do NOT coerce strings (unlike arithmetic).
-- All of these should raise errors, not silently coerce.

-- BAND must reject strings
local ok, err = pcall(function() return "10" & 0xff end)
assert(not ok, "expected error for string & int, got: " .. tostring(ok))
assert(string.find(err, "attempt to perform bitwise operation on a string value"),
  "expected bitwise error, got: " .. tostring(err))

-- BOR must reject strings
ok, err = pcall(function() return "10" | 0 end)
assert(not ok, "expected error for string | int")

-- BXOR must reject strings
ok, err = pcall(function() return "10" ~ 0 end)
assert(not ok, "expected error for string ~ int")

-- BNOT must reject strings
ok, err = pcall(function() return ~"10" end)
assert(not ok, "expected error for ~string")

-- SHL must reject strings
ok, err = pcall(function() return "10" << 1 end)
assert(not ok, "expected error for string << int")

-- SHR must reject strings
ok, err = pcall(function() return "10" >> 1 end)
assert(not ok, "expected error for string >> int")

-- Also test string on the right side
ok, err = pcall(function() return 0xff & "10" end)
assert(not ok, "expected error for int & string")

ok, err = pcall(function() return 1 << "2" end)
assert(not ok, "expected error for int << string")

ok, err = pcall(function() return 8 >> "1" end)
assert(not ok, "expected error for int >> string")

