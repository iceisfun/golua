-- Test: goto.lua - Long labels
-- From: goto.lua
-- What: Tests that labels with very long names (400 characters) work correctly

do
  local prog = [[
  do
    local a = 1
    goto l%sa; a = a + 1
   ::l%sa:: a = a + 10
    goto l%sb; a = a + 2
   ::l%sb:: a = a + 20
    return a
  end
  ]]
  local label = string.rep("0123456789", 40)
  prog = string.format(prog, label, label, label, label)
  assert(assert(load(prog))() == 31)
end
