-- Concatenation error messages should identify the non-concatenable operand,
-- not the valid one.
--
-- When "hello" .. nil fails, the error should say "nil value" (the problem)
-- not "string value" (which is fine for concatenation).

-- Test 1: string .. nil → should mention "nil"
local ok1, err1 = pcall(function() return "hello" .. nil end)
assert(not ok1)
assert(string.find(err1, "nil value"),
  "should mention 'nil value', got: " .. err1)

-- Test 2: string .. boolean → should mention "boolean"
local ok2, err2 = pcall(function() return "hello" .. true end)
assert(not ok2)
assert(string.find(err2, "boolean value"),
  "should mention 'boolean value', got: " .. err2)

-- Test 3: number .. table → should mention "table"
local ok3, err3 = pcall(function() return 42 .. {} end)
assert(not ok3)
assert(string.find(err3, "table value"),
  "should mention 'table value', got: " .. err3)

-- Test 4: number .. function → should mention "function"
local ok4, err4 = pcall(function() return 42 .. print end)
assert(not ok4)
assert(string.find(err4, "function value"),
  "should mention 'function value', got: " .. err4)

-- Test 5: nil .. string → should mention "nil" (left operand bad)
local ok5, err5 = pcall(function() return nil .. "hello" end)
assert(not ok5)
assert(string.find(err5, "nil value"),
  "should mention 'nil value', got: " .. err5)

-- Test 6: valid concatenations still work
assert("hello" .. " " .. "world" == "hello world")
assert(42 .. "" == "42")
assert("" .. 3.14 == "3.14")

print("PASS")
