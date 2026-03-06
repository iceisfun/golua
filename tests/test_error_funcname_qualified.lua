-- BUG: Error messages for stdlib functions use unqualified names
--
-- Reference Lua uses qualified names like 'string.char', 'math.abs',
-- 'table.concat' in error messages. GoLua uses just 'char', 'abs', 'concat'.
--
-- Also: Reference Lua says "got no value" when no argument is passed,
-- while GoLua says "got nil".

local function check_err(fn, args, expected_pattern)
  local ok, err = pcall(fn, table.unpack(args))
  assert(not ok, "expected error")
  assert(string.find(err, expected_pattern, 1, true),
    string.format("expected '%s' in error, got: %s", expected_pattern, err))
end

-- Qualified function names in errors
check_err(string.char, {256}, "'string.char'")
check_err(math.abs, {}, "'math.abs'")
check_err(table.concat, {"x"}, "'table.concat'")
check_err(table.insert, {}, "'table.insert'")

-- "got no value" vs "got nil"
local ok, err = pcall(math.abs)
assert(string.find(err, "got no value", 1, true),
  "expected 'got no value' in: " .. err)

print("PASS")
