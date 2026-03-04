-- Test: nextvar.lua - Empty loops and single-iteration loops
-- From: nextvar.lua
-- What: Tests that for-in with empty tables, numeric for with zero iterations, and single-iteration loops work correctly.

do
  for a,b in pairs{} do error"not here" end
  for i=1,0 do error'not here' end
  for i=0,1,-1 do error'not here' end
  local a = nil; for i=1,1 do assert(not a); a=1 end; assert(a)
  a = nil; for i=1,1,-1 do assert(not a); a=1 end; assert(a)
end
