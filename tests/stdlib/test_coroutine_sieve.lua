-- Test: coroutine.lua - Sieve of Eratosthenes with coroutines
-- From: coroutine.lua
-- What: Tests coroutine-based prime sieve (functional coroutine pipeline)

do
  local function gen (n)
    return coroutine.wrap(function ()
      for i=2,n do coroutine.yield(i) end
    end)
  end

  local function filter (p, g)
    return coroutine.wrap(function ()
      while 1 do
        local n = g()
        if n == nil then return end
        if math.fmod(n, p) ~= 0 then coroutine.yield(n) end
      end
    end)
  end

  local x = gen(80)
  local a = {}
  while 1 do
    local n = x()
    if n == nil then break end
    table.insert(a, n)
    x = filter(n, x)
  end
  assert(#a == 22 and a[#a] == 79)
end
