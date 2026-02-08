-- BROKEN: Bitwise binary metamethods not checked
-- __band, __bor, __bxor, __shl, __shr should be invoked when operands
-- cannot be converted to integers, but currently bitwise ops error directly
-- without metamethod lookup.

-- __band
local mt_band = { __band = function(a, b) return "band" end }
local t1 = setmetatable({}, mt_band)
assert((t1 & 1) == "band", "__band metamethod should be invoked")

-- __bor
local mt_bor = { __bor = function(a, b) return "bor" end }
local t2 = setmetatable({}, mt_bor)
assert((t2 | 1) == "bor", "__bor metamethod should be invoked")

-- __bxor
local mt_bxor = { __bxor = function(a, b) return "bxor" end }
local t3 = setmetatable({}, mt_bxor)
assert((t3 ~ 1) == "bxor", "__bxor metamethod should be invoked")

-- __shl
local mt_shl = { __shl = function(a, b) return "shl" end }
local t4 = setmetatable({}, mt_shl)
assert((t4 << 1) == "shl", "__shl metamethod should be invoked")

-- __shr
local mt_shr = { __shr = function(a, b) return "shr" end }
local t5 = setmetatable({}, mt_shr)
assert((t5 >> 1) == "shr", "__shr metamethod should be invoked")
