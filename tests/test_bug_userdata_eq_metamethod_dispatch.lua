-- Bug: __eq metamethod was not invoked for userdata comparisons.
-- Lua 5.4: when raw equality fails and both operands are userdata/tables,
-- VM checks left __eq first, then right.

local function new_ud()
  return assert(io.tmpfile())
end

-- Different metatables, both with __eq
do
  local a = new_ud()
  local b = new_ud()
  debug.setmetatable(a, {__eq = function() return true end})
  debug.setmetatable(b, {__eq = function() return true end})
  assert(a == b, "userdata __eq with different metatables should fire")
  assert(not (a ~= b), "userdata ~= should mirror __eq")
end

-- Left operand precedence
do
  local left = new_ud()
  local right = new_ud()
  debug.setmetatable(left, {__eq = function() return true end})
  debug.setmetatable(right, {__eq = function() return false end})
  assert(left == right, "left userdata __eq should win")
  assert(right ~= left, "right userdata __eq should win when operand order flips")
end

-- One side with __eq should still be used
do
  local has_eq = new_ud()
  local no_eq = new_ud()
  debug.setmetatable(has_eq, {__eq = function() return true end})
  debug.setmetatable(no_eq, {})
  assert(has_eq == no_eq, "left-only userdata __eq should fire")
  assert(no_eq == has_eq, "right-only userdata __eq should fire")
end

-- Raw-equal userdata must skip __eq
do
  local same = new_ud()
  debug.setmetatable(same, {__eq = function() error("should not be called") end})
  assert(same == same, "raw-equal userdata must skip __eq")
end

-- Mixed table/userdata should not trigger __eq; result is false
do
  local ud = new_ud()
  local t = {}
  debug.setmetatable(ud, {__eq = function() return true end})
  setmetatable(t, {__eq = function() return true end})
  assert(not (ud == t), "userdata vs table should be raw-unequal")
  assert(ud ~= t, "userdata vs table should be ~= true")
end
