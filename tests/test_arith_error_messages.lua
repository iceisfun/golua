-- BUG: Arithmetic error messages use generic format instead of specific operation names
-- Reference Lua 5.4 says: "attempt to add a 'string' with a 'number'"
-- GoLua says:              "attempt to perform arithmetic on a string value"
-- The specific operation name (add/sub/mul/div/mod/pow/idiv/unm) and both types should be shown.

local function check(expected_pattern, fn)
  local ok, err = pcall(fn)
  assert(not ok, "should error")
  assert(err:find(expected_pattern),
    "expected pattern '" .. expected_pattern .. "' but got: " .. tostring(err))
end

-- Binary operations with string that can't be coerced
check("attempt to add", function() return "hello" + 1 end)
check("attempt to sub", function() return "hello" - 1 end)
check("attempt to mul", function() return "hello" * 1 end)
check("attempt to div", function() return "hello" / 1 end)
check("attempt to mod", function() return "hello" % 1 end)
check("attempt to pow", function() return "hello" ^ 1 end)
check("attempt to idiv", function() return "hello" // 1 end)

-- Unary minus
check("attempt to unm", function() return -"hello" end)

-- Should also include both types in quotes
local ok, err = pcall(function() return "hello" + 1 end)
assert(err:find("'string'"), "should mention 'string' in quotes, got: " .. err)
assert(err:find("'number'"), "should mention 'number' in quotes, got: " .. err)

-- Two strings that fail
local ok2, err2 = pcall(function() return "abc" + "def" end)
assert(err2:find("attempt to add"), "got: " .. err2)
assert(err2:find("'string'"), "should mention 'string', got: " .. err2)

