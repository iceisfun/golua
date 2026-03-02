-- Test: nextvar.lua - checknext consistency
-- From: nextvar.lua
-- What: Tests that manual next-based iteration and pairs-based iteration produce identical results for tables with mixed array and hash parts.

do
  local function checknext (a)
    local b = {}
    do local k,v = next(a); while k do b[k] = v; k,v = next(a,k) end end
    for k,v in pairs(b) do assert(a[k] == v) end
    for k,v in pairs(a) do assert(b[k] == v) end
  end

  checknext{1,x=1,y=2,z=3}
  checknext{1,2,x=1,y=2,z=3}
  checknext{1,2,3,x=1,y=2,z=3}
  checknext{1,2,3,4,x=1,y=2,z=3}
  checknext{1,2,3,4,5,x=1,y=2,z=3}
end
