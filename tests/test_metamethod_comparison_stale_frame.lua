-- Bug: OP_EQ/OP_LT/OP_LE use stale frame pointer after metamethod calls.
-- When a metamethod triggers callStack growth (via append), the frame pointer
-- taken at the top of the execute() loop becomes invalid. frame.pc++ writes
-- to freed memory, so the JMP skip is lost and comparison returns wrong result.

-- __eq metamethod with a function call inside (forces callStack growth)
local function empty() end
local mt = {__eq = function(a, b) empty(); return true end}
local t1 = setmetatable({}, mt)
local t2 = setmetatable({}, mt)
assert(t1 == t2, "__eq metamethod result lost due to stale frame pointer")
assert(not (t1 ~= t2), "__eq ~= should negate metamethod result")

-- __lt metamethod with nested call
local mt2 = {__lt = function(a, b) empty(); return true end}
local a = setmetatable({}, mt2)
local b = setmetatable({}, mt2)
assert(a < b, "__lt metamethod result lost due to stale frame pointer")

-- __le metamethod with nested call
local mt3 = {__le = function(a, b) empty(); return true end}
local c = setmetatable({}, mt3)
local d = setmetatable({}, mt3)
assert(c <= d, "__le metamethod result lost due to stale frame pointer")

-- __eq that returns false should also work correctly
local mt4 = {__eq = function(a, b) empty(); return false end}
local e = setmetatable({}, mt4)
local f = setmetatable({}, mt4)
assert(not (e == f), "__eq returning false should mean not equal")
assert(e ~= f, "__eq returning false should mean ~=")

print("PASS")
