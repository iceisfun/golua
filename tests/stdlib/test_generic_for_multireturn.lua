-- Test: nextvar.lua - Generic for with multiple return values
-- From: nextvar.lua
-- What: Tests generic for-in loop with an iterator that returns multiple values, verifying they are correctly unpacked.

do
  local function f (n, p)
    local t = {}; for i=1,p do t[i] = i*10 end
    return function (_, n, ...)
             assert(select("#", ...) == 0)  -- no extra arguments
             if n > 0 then
               n = n-1
               return n, table.unpack(t)
             end
           end, nil, n
  end

  local x = 0
  for n,a,b,c,d in f(5,3) do
    x = x+1
    assert(a == 10 and b == 20 and c == 30 and d == nil)
  end
  assert(x == 5)
end
