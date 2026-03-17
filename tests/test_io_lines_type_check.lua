-- io.lines() with non-string first argument should error.
-- In lua5.4, io.lines(true) and io.lines({}) raise type errors.

-- io.lines with boolean should error
local ok1, err1 = pcall(io.lines, true)
assert(not ok1, "io.lines(true) should error, got function")

-- io.lines with table should error
local ok2, err2 = pcall(io.lines, {})
assert(not ok2, "io.lines({}) should error, got function")
