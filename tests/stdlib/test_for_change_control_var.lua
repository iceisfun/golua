-- Test: Lua 5.5 - for-loop control variable is read-only
-- From: nextvar.lua (updated for Lua 5.5)
-- What: Tests that assigning to the loop control variable is a compile error in Lua 5.5.

-- In Lua 5.5, assigning to a for-loop control variable is a compile-time error.
do
  local f, err = load("for i = 1, 10 do i = 'x' end")
  assert(f == nil)
  assert(string.find(err, "attempt to assign to const variable 'i'"))
end

-- Verify iteration still works correctly without assignment to control variable
do
  local a = 0
  for i = 1, 10 do a = a + 1 end
  assert(a == 10)

  a = 0
  for i = 10.0, 1, -1 do a = a + 1 end
  assert(a == 10)
end
