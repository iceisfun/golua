-- Tests for round 14 bugs 4, 5, and 6

-- Bug 4: Modulo constant folding loses negative zero sign
-- -0.0 % 1 should produce -0.0, so 1 / (-0.0 % 1) should be -inf
assert(1 / (-0.0 % 1) == -math.huge, "constant folded -0.0 % 1 should be -0.0 (got " .. tostring(1 / (-0.0 % 1)) .. ")")

-- Also test via a variable to make sure runtime agrees
local z = -0.0
assert(1 / (z % 1) == -math.huge, "runtime -0.0 % 1 should be -0.0")

-- Bug 5 (Lua 5.4): `local <attr> name` syntax was rejected.
-- In Lua 5.5, `local<const>` and `local <close>` are valid prefix-attribute syntax.
local f = load("local <close> x = nil")
assert(f ~= nil, "local <close> x should be valid in 5.5")

local f2 = load("local <const> x = 1")
assert(f2 ~= nil, "local <const> x should be valid in 5.5")

-- Both prefix-attribute and post-attribute syntax should work
local f3 = load("local x <close> = setmetatable({}, {__close=function()end})")
assert(f3 ~= nil, "valid local x <close> should parse")

local f4 = load("local x <const> = 1")
assert(f4 ~= nil, "valid local x <const> should parse")

-- Bug 6: `for` loop single-name error message
local f5, err5 = load("for i")
assert(f5 == nil)
assert(err5:find("'=' or 'in' expected"), "error should say '=' or 'in' expected, got: " .. tostring(err5))

-- Also test with unexpected token (not just EOF)
local f6, err6 = load("for i +")
assert(f6 == nil)
assert(err6:find("'=' or 'in' expected"), "error should say '=' or 'in' expected, got: " .. tostring(err6))

print("PASS")
