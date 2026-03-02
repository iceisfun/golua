-- Test: closure.lua - Closure over loop variable with GC
-- From: closure.lua
-- What: Tests closures capturing loop variables, with garbage collection between calls

do
  local A,B = 0,{g=10}
  local function f(x)
    local a = {}
    for i=1,1000 do
      local y = 0
      do
        a[i] = function () B.g = B.g+1; y = y+x; return y+A end
      end
    end
    local dummy = function () return a[A] end
    collectgarbage()
    A = 1; assert(dummy() == a[1]); A = 0;
    assert(a[1]() == x)
    assert(a[3]() == x)
    collectgarbage()
    assert(B.g == 12)
    return a
  end
  local a = f(10)
end
