-- test_string_format_c_contract: string.format('%c', ...) should match Lua 5.4

-- Integer and integer-representable values are accepted
assert(string.format("%c", 65) == "A", "%c with int should work")
assert(string.format("%c", 65.0) == "A", "%c with integer float should work")

-- Fractional values must error
local ok1, err1 = pcall(function() return string.format("%c", 65.1) end)
assert(ok1 == false, "%c with fractional float should error")
assert(type(err1) == "string" and err1:find("number has no integer representation"),
       "unexpected %c fractional error: " .. tostring(err1))

local ok2, err2 = pcall(function() return string.format("%c", "65.1") end)
assert(ok2 == false, "%c with fractional numeric string should error")
assert(type(err2) == "string" and err2:find("number has no integer representation"),
       "unexpected %c fractional-string error: " .. tostring(err2))

-- %c writes a single byte (wrap modulo 256)
assert(string.byte(string.format("%c", 255)) == 255, "%c 255 should produce byte 255")
assert(string.byte(string.format("%c", 256)) == 0, "%c 256 should wrap to byte 0")
assert(string.byte(string.format("%c", -1)) == 255, "%c -1 should wrap to byte 255")
