-- Test: events.lua - rawlen
-- From: events.lua
-- What: Tests rawlen() bypasses __len metamethod

do
  local t = setmetatable({1,2,3}, {__len = function () return 10 end})
  assert(#t == 10 and rawlen(t) == 3)
  assert(rawlen"abc" == 3)
  assert(not pcall(rawlen, io.stdin))
  assert(not pcall(rawlen, 34))
  assert(not pcall(rawlen))

  -- rawlen for long strings
  assert(rawlen(string.rep('a', 1000)) == 1000)
end
