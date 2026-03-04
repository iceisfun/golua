-- Test: events.lua - rawlen
-- From: events.lua
-- What: Tests rawlen() bypasses __len metamethod

do
  local t = setmetatable({1,2,3}, {__len = function () return 10 end})
  assert(#t == 10 and rawlen(t) == 3)
  assert(rawlen"abc" == 3)
  -- In C Lua, io.stdin is userdata so rawlen errors; in GoLua it's a table
  if io and io.stdin then
    -- GoLua: io.stdin is a table, rawlen succeeds (returns 0)
    assert(type(io.stdin) == "table" or not pcall(rawlen, io.stdin))
  end
  assert(not pcall(rawlen, 34))
  assert(not pcall(rawlen))

  -- rawlen for long strings
  assert(rawlen(string.rep('a', 1000)) == 1000)
end
