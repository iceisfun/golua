-- Bug: __eq metamethod not invoked when two tables have different metatables.
-- Lua 5.4: __eq is tried when both operands are tables (or both userdata) and
-- raw equality fails. The left operand's metamethod is checked first.

-- Different metatables, both with __eq
local a = setmetatable({}, {__eq = function() return true end})
local b = setmetatable({}, {__eq = function() return true end})
assert(a == b, "__eq with different metatables should fire (left's __eq returns true)")
assert(not (a ~= b), "__eq ~= inverse should also work")

-- Left operand's __eq takes precedence
local left = setmetatable({}, {__eq = function() return true end})
local right = setmetatable({}, {__eq = function() return false end})
assert(left == right, "left __eq should win: returns true")
assert(right ~= left, "when right is left operand, its __eq (false) should be used")

-- Only one side has __eq
local has_eq = setmetatable({}, {__eq = function() return true end})
local no_eq = setmetatable({}, {})
assert(has_eq == no_eq, "one side has __eq: should fire")
assert(no_eq == has_eq, "other side has __eq: should fire")

-- Same metatable (guard: should already work)
local shared_mt = {__eq = function() return true end}
local c = setmetatable({}, shared_mt)
local d = setmetatable({}, shared_mt)
assert(c == d, "same metatable __eq should work")

-- __eq returning false
local e = setmetatable({}, {__eq = function() return false end})
local f = setmetatable({}, {__eq = function() return false end})
assert(not (e == f), "__eq returning false: == should be false")
assert(e ~= f, "__eq returning false: ~= should be true")

-- __eq not called for non-table types (guard)
assert(not (1 == "1"), "no __eq for int vs string")
assert(not (nil == false), "no __eq for nil vs false")

-- Raw equal tables should NOT invoke __eq
local same = setmetatable({}, {__eq = function() error("should not be called") end})
assert(same == same, "raw-equal tables skip __eq")
