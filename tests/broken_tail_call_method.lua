-- Test: calls.lua - Method tail calls
-- From: calls.lua
-- What: Tests that method calls in return position are compiled as tail calls
-- Status: BROKEN - compiler generates OP_CALL instead of OP_TAILCALL for self:method() calls

do
  local a = {}
  function a:deep (n) if n>0 then return self:deep(n-1) else return 101 end end
  assert(a:deep(30000) == 101)
end
