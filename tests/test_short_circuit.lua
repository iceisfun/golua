-- Test short-circuit evaluation: and/or ternary idiom

-- Basic and/or ternary: true condition
local x = true and "yes" or "no"
assert(x == "yes", "true and 'yes' or 'no' should be 'yes', got: " .. tostring(x))

-- Basic and/or ternary: false condition
local y = false and "yes" or "no"
assert(y == "no", "false and 'yes' or 'no' should be 'no', got: " .. tostring(y))

-- nil condition falls through to or-branch
local z = nil and "yes" or "no"
assert(z == "no", "nil and 'yes' or 'no' should be 'no', got: " .. tostring(z))

-- Non-nil comparison with function call returning table (original bug scenario)
local id = "foo"

function get_def()
  return {}
end

local def = id ~= nil and get_def() or nil
assert(def ~= nil, "id ~= nil and get_def() or nil should return table, got nil")
assert(type(def) == "table", "expected table, got: " .. type(def))

-- Same pattern when id is nil
id = nil
def = id ~= nil and get_def() or nil
assert(def == nil, "nil id should produce nil def")

-- and returns first falsy or last value
assert((1 and 2) == 2)
assert((1 and false) == false)
assert((1 and nil) == nil)
assert((false and 2) == false)
assert((nil and 2) == nil)
assert((1 and 2 and 3) == 3)
assert((1 and false and 3) == false)
assert((1 and nil and 3) == nil)

-- or returns first truthy or last value
assert((1 or 2) == 1)
assert((false or 2) == 2)
assert((nil or 2) == 2)
assert((false or nil) == nil)
assert((nil or false) == false)
assert((false or nil or 3) == 3)
assert((nil or false or 3) == 3)

-- Ternary with function calls (side effects)
local call_count = 0
local function side_effect()
  call_count = call_count + 1
  return "called"
end

-- true condition: or-branch should NOT be evaluated
call_count = 0
local r = true and "yes" or side_effect()
assert(r == "yes")
assert(call_count == 0, "or-branch should not execute when and-branch is truthy")

-- false condition: and-branch short-circuits, or-branch runs
call_count = 0
r = false and side_effect() or "fallback"
assert(r == "fallback")
assert(call_count == 0, "and-branch should not evaluate second operand when first is falsy")

-- Nested ternary
local a = true and (false and "inner" or "outer") or "none"
assert(a == "outer", "nested ternary failed, got: " .. tostring(a))

-- Ternary with 0 (truthy in Lua, unlike C)
local b = 0 and "zero is truthy" or "zero is falsy"
assert(b == "zero is truthy", "0 should be truthy in Lua")

-- Ternary with empty string (truthy in Lua)
local c = "" and "empty is truthy" or "empty is falsy"
assert(c == "empty is truthy", "empty string should be truthy in Lua")

-- Ternary inside function, returning result
local function ternary(cond, t, f)
  return cond and t or f
end
assert(ternary(true, 10, 20) == 10)
assert(ternary(false, 10, 20) == 20)
assert(ternary("yes", "a", "b") == "a")
assert(ternary(nil, "a", "b") == "b")

-- WARNING: and/or ternary fails when middle value is falsy
-- This is expected Lua behavior, not a bug
local gotcha = true and false or "oops"
assert(gotcha == "oops", "and/or ternary pitfall: true and false or 'oops' should be 'oops'")

local gotcha2 = true and nil or "oops"
assert(gotcha2 == "oops", "and/or ternary pitfall: true and nil or 'oops' should be 'oops'")
