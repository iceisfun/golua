-- Test: string.format %s with nil/bool and __tostring
-- From: strings.lua
-- What: Tests string.format %s with nil, booleans, precision truncation, and objects with __tostring/__name metamethods. Also tests error when __tostring returns non-string.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

assert(string.format("\0%s\0", "\0\0\1") == "\0\0\0\1\0")
checkerror("contains zeros", string.format, "%10s", "\0")

-- format x tostring
assert(string.format("%s %s", nil, true) == "nil true")
assert(string.format("%s %.4s", false, true) == "false true")
assert(string.format("%.3s %.3s", false, true) == "fal tru")
local m = setmetatable({}, {__tostring = function () return "hello" end,
                            __name = "hi"})
assert(string.format("%s %.10s", m, m) == "hello hello")
getmetatable(m).__tostring = nil   -- will use '__name' from now on
assert(string.format("%.4s", m) == "hi: ")

getmetatable(m).__tostring = function () return {} end
checkerror("'__tostring' must return a string", tostring, m)
end
