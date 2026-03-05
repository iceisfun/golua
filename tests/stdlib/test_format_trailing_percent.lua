-- A trailing % in a format string should error, not silently output "%".
-- Lua 5.4 behavior: checks for argument first, then errors on invalid conversion.

-- No extra args: "no value" error
local ok, err = pcall(string.format, "%")
assert(not ok, "trailing % should error")
assert(tostring(err):find("no value"), "expected 'no value' error, got: " .. tostring(err))

-- With trailing text before %
local ok2, err2 = pcall(string.format, "hello%")
assert(not ok2, "trailing % after text should error")
assert(tostring(err2):find("no value"), "expected 'no value' error, got: " .. tostring(err2))

-- With extra arg: "invalid conversion" error
local ok3, err3 = pcall(string.format, "%", 1)
assert(not ok3, "trailing % with arg should error")
assert(tostring(err3):find("invalid conversion"), "expected 'invalid conversion' error, got: " .. tostring(err3))

-- %% at end is fine (literal percent)
assert(string.format("%%") == "%")
assert(string.format("hello%%") == "hello%")

-- % followed by flags but no conversion
local ok4, err4 = pcall(string.format, "%0")
assert(not ok4, "% with flag but no conversion should error")
assert(tostring(err4):find("no value"), "expected 'no value' error, got: " .. tostring(err4))

local ok5, err5 = pcall(string.format, "%0", 1)
assert(not ok5, "% with flag but no conversion (with arg) should error")
assert(tostring(err5):find("invalid conversion"), "expected 'invalid conversion', got: " .. tostring(err5))

print("OK")
