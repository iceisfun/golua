-- Bug: string.format doesn't validate excessive width specifiers.
-- Lua 5.4: widths >= 100 (or precision >= 100) raise an error:
-- "invalid format (width or precision too long)"

-- Excessive width should error
local ok1, e1 = pcall(string.format, "%999999d", 1)
assert(not ok1, "format %999999d should error")

local ok2, e2 = pcall(string.format, "%200d", 1)
assert(not ok2, "format %200d should error")

-- Excessive precision should error
local ok3, e3 = pcall(string.format, "%.200f", 1.0)
assert(not ok3, "format %.200f should error")

-- Guard: valid widths should still work
assert(string.format("%10d", 42) == "        42")
assert(string.format("%-10d", 42) == "42        ")
assert(string.format("%05d", 42) == "00042")
assert(string.format("%.5f", 3.14) == "3.14000")

-- Lua 5.4 limit is 99
local ok4, e4 = pcall(string.format, "%99d", 1)
assert(ok4, "format %99d should be ok")
local ok5, e5 = pcall(string.format, "%100d", 1)
assert(not ok5, "format %100d should error")
