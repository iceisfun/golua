-- Test: nextvar.lua - Changing control variable in numeric for
-- From: nextvar.lua
-- What: Tests that assigning to the loop control variable inside a numeric for body does not affect iteration.

do   -- changing the control variable
  local a
  a = 0; for i = 1, 10 do a = a + 1; i = "x" end; assert(a == 10)
  a = 0; for i = 10.0, 1, -1 do a = a + 1; i = "x" end; assert(a == 10)
end
