-- Test: goto.lua - Jump over local declaration to end of block
-- From: goto.lua
-- What: Tests that jumping over a local declaration is valid when the target is at the end of the block

do
  local x = 13  -- initialized for the final assertion

  do
    goto l1
    local a = 23
    x = a
    ::l1::;
  end

  while true do
    goto l4
    goto l1
    goto l1
    local xx = 45
    ::l1:: ;;;
  end
  ::l4:: assert(x == 13)

  if print then
    goto l1
    error("should not be here")
    goto l2
    local xx
    ::l1:: ; ::l2:: ;;
  else end
end
