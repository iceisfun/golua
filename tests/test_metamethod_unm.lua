-- BROKEN: __unm metamethod not checked in OP_UNM
-- The unary minus operator should fall back to __unm when the operand
-- is not a number, but currently it errors directly without metamethod lookup.

local mt = { __unm = function(a) return 42 end }
local t = setmetatable({}, mt)
assert(-t == 42, "__unm metamethod should be invoked for non-number operand")
