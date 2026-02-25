-- Bug #6: math.type returns false for non-numbers instead of nil (Lua 5.4 standard)
-- Lua 5.4's luaL_pushfail is lua_pushnil, so math.type returns nil for non-numbers.

assert(math.type("x") == nil, "math.type on string should be nil, not false")
assert(math.type(true) == nil, "math.type on boolean should be nil, not false")
assert(math.type(nil) == nil, "math.type on nil should be nil, not false")
assert(math.type({}) == nil, "math.type on table should be nil, not false")

-- Verify false vs nil distinction
assert(math.type("x") ~= false, "math.type should return nil, not false")

-- Number types should still work
assert(math.type(1) == "integer")
assert(math.type(1.0) == "float")
