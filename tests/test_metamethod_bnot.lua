-- BROKEN: __bnot metamethod not checked in OP_BNOT
-- The bitwise NOT operator should fall back to __bnot when the operand
-- cannot be converted to integer, but currently it errors directly.

local mt = { __bnot = function(a) return 42 end }
local t = setmetatable({}, mt)
assert(~t == 42, "__bnot metamethod should be invoked for non-integer operand")
