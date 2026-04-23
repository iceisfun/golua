-- Lua 5.4/5.5 removed the implicit __le -> not (b < a) fallback.
-- With only __lt defined, a <= b (and b >= a) between tables must raise
-- "attempt to compare two table values" rather than silently calling __lt.
-- See Lua 5.5 reference manual section 2.4.

-- Case 1: only __lt defined on both tables; `a <= b` must error.
local a = setmetatable({ id = "a" }, {
    __lt = function(x, y) return true end,
})
local b = setmetatable({ id = "b" }, {
    __lt = function(x, y) return false end,
})

local ok, err = pcall(function() return a <= b end)
assert(not ok, "a <= b must error when only __lt is defined")
assert(string.find(err, "attempt to compare two table values", 1, true),
    "expected 'attempt to compare two table values', got: " .. tostring(err))

-- Case 2: mirror `>=` (compiled as reversed `<=`) must also error.
local ok2, err2 = pcall(function() return a >= b end)
assert(not ok2, "a >= b must error when only __lt is defined")
assert(string.find(err2, "attempt to compare two table values", 1, true),
    "expected 'attempt to compare two table values', got: " .. tostring(err2))

-- Case 3: no metamethods at all on tables; plain error message.
local p, q = {}, {}
local ok3, err3 = pcall(function() return p <= q end)
assert(not ok3, "p <= q between bare tables must error")
assert(string.find(err3, "attempt to compare two table values", 1, true),
    "expected 'attempt to compare two table values', got: " .. tostring(err3))

-- Case 4: __le IS defined -> must still work (not regressed).
local c = setmetatable({}, { __le = function(x, y) return true end })
local d = setmetatable({}, {})
assert(c <= d, "__le must still be honored when defined")
assert(d >= c, "__le must still be honored for mirrored >=")

-- Case 5: __le returning false also works.
local e = setmetatable({}, { __le = function(x, y) return false end })
local f = setmetatable({}, {})
assert(not (e <= f), "__le returning false must be honored")

-- Case 6: only right operand provides __le -- still honored.
local g = setmetatable({}, {})
local h = setmetatable({}, { __le = function(x, y) return true end })
assert(g <= h, "right operand's __le must be used when left has none")
