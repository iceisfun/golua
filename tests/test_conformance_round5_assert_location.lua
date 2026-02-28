-- BUG: assert() doesn't prepend file:line to string error messages
-- In Lua 5.4, when assert fails with a string message, the error
-- has file:line prepended (like error() with default level).
-- GoLua returns the bare message without location info.

-- assert with string message should have location prefix
local ok, err = pcall(function() assert(false, "oops") end)
assert(not ok, "assert(false) should error")
assert(type(err) == "string", "assert error should be string")
assert(err:find(": oops"),
    "assert error should contain ': oops' (with location prefix), got: " .. err)

-- assert with no message should also have location
local ok2, err2 = pcall(function() assert(false) end)
assert(not ok2, "assert(false) should error")
assert(type(err2) == "string", "assert error should be string")
assert(err2:find(": assertion failed!"),
    "assert error should contain ': assertion failed!' (with location prefix), got: " .. err2)

-- assert with non-string message should NOT add location
local ok3, err3 = pcall(function() assert(false, 42) end)
assert(not ok3, "assert(false, 42) should error")
assert(err3 == 42, "assert with number message should preserve the number")

-- assert with table message should NOT add location
local ok4, err4 = pcall(function() assert(false, {msg="bad"}) end)
assert(not ok4, "assert(false, table) should error")
assert(type(err4) == "table", "assert with table message should preserve the table")
assert(err4.msg == "bad", "assert table error should be intact")
