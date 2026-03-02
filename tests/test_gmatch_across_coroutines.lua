-- Test: gmatch iterator across coroutines
-- From: strings.lua
-- What: Tests that a string.gmatch iterator works correctly when resumed across different coroutines (bug fix from Lua 5.3.2).

do
  local f = string.gmatch("1 2 3 4 5", "%d+")
  assert(f() == "1")
  local co = coroutine.wrap(f)
  assert(co() == "2")
end
