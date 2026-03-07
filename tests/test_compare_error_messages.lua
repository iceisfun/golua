-- BUG: Same-type comparison errors say "TYPE with TYPE" instead of "two TYPE values"
-- Reference Lua 5.4: "attempt to compare two table values"
-- GoLua:             "attempt to compare table with table"

local function check(expected_pattern, fn)
  local ok, err = pcall(fn)
  assert(not ok, "should error")
  assert(err:find(expected_pattern),
    "expected pattern '" .. expected_pattern .. "' but got: " .. tostring(err))
end

-- Same-type comparisons should use "two TYPE values" format
check("two table values", function() return {} < {} end)
check("two table values", function() return {} <= {} end)
check("two boolean values", function() return true < false end)
check("two nil values", function() return nil < nil end)
check("two function values", function()
  return (function() end) < (function() end)
end)

-- Different-type comparisons should use "TYPE with TYPE" format (already correct)
check("number with string", function() return 1 < "hello" end)
check("string with table", function() return "x" < {} end)

