-- Regression: os.date argument errors must resolve the funcname dynamically
-- (short 'date' on a direct call, qualified 'os.date' via pcall), matching
-- reference Lua's luaL_argerror, instead of a hard-coded 'os.date' literal.

-- Direct call: bytecode debug info yields the short field name 'date'.
local ok1, e1 = pcall(function() return os.date("!%4Y", 0) end)
assert(not ok1)
assert(e1:find("to 'date'"), "arg#1 specifier error should say 'date', got: " .. tostring(e1))
assert(e1:find("invalid conversion specifier"), "should mention specifier, got: " .. tostring(e1))

local ok2, e2 = pcall(function() return os.date("!%Y", {}) end)
assert(not ok2)
assert(e2:find("to 'date'"), "arg#2 type error should say 'date', got: " .. tostring(e2))
assert(e2:find("#2"), "should be arg #2, got: " .. tostring(e2))
assert(e2:find("number expected, got table"), "should report table, got: " .. tostring(e2))

-- Indirect via pcall(os.date, ...): global lookup yields qualified 'os.date'.
local ok3, e3 = pcall(os.date, "!%4Y", 0)
assert(not ok3)
assert(e3:find("to 'os%.date'"), "pcall'd call should say 'os.date', got: " .. tostring(e3))

print("PASS")
