-- Unary minus should coerce strings to numbers (Lua 5.4)
assert(-"2" == -2, "unary minus string coercion")
assert(-"3.5" == -3.5, "unary minus float string coercion")
assert(not pcall(function() return -"bob" end), "unary minus non-numeric should error")
