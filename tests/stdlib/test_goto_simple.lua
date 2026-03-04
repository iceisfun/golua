-- Test: goto.lua - Simple gotos
-- From: goto.lua
-- What: Tests basic forward and backward goto jumps

do
  local x
  do
    local y = 12
    goto l1
    ::l2:: x = x + 1; goto l3
    ::l1:: x = y; goto l2
  end
  ::l3:: ::l3_1:: assert(x == 13)
end
