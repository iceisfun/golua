-- Test: nextvar.lua - ipairs with __index metamethod
-- From: nextvar.lua
-- What: Tests that ipairs works with __index metamethod to provide virtual array elements.

do
  local a = {n=10}
  setmetatable(a, { __index = function (t,k)
                       if k <= t.n then return k * 10 end
                    end})
  local i = 0
  for k,v in ipairs(a) do
    i = i + 1
    assert(k == i and v == i * 10)
  end
  assert(i == a.n)
end
