-- Test: goto.lua - Bug in 5.3 goto and if-then
-- From: goto.lua
-- What: Tests correct control flow with goto inside an if-then block jumping backward

do
  local first = true
  local a = false
  if true then
    goto LBL
    ::loop::
    a = true
    ::LBL::
    if first then
      first = false
      goto loop
    end
  end
  assert(a)
end
