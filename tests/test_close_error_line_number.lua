-- __close error should report the line where the scope ends (where __close is invoked),
-- not the line where the to-be-closed variable is declared.
-- Lua 5.4 reports the scope-end line.

local ok, err = pcall(function()
  local x <close> = setmetatable({}, {__close = "bad"})
end)
-- The error should reference line 7 (the 'end'), not line 6 (the declaration)
assert(type(err) == "string")
local line = err:match(":(%d+):")
assert(line == "7", "expected error on line 7 (scope end), got line " .. tostring(line))

print("OK")
