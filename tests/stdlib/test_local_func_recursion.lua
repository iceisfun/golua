-- Test: calls.lua - Local function recursion
-- From: calls.lua
-- What: Tests that local functions can be recursive and don't leak into outer scope

do
  fact = false
  do
    local res = 1
    local function fact (n)
      if n==0 then return res
      else return n*fact(n-1)
      end
    end
    assert(fact(5) == 120)
  end
  assert(fact == false)
end
