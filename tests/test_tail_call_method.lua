-- Test: calls.lua - Method tail calls
-- From: calls.lua
-- What: Tests that method calls in return position are compiled as tail calls

-- Basic method tail call with deep recursion
do
  local a = {}
  function a:deep(n) if n > 0 then return self:deep(n - 1) else return 101 end end
  assert(a:deep(30000) == 101)
end

-- Inherited method tail call via metatable chain
do
  local proto = {}
  function proto:deep(n)
    if n > 0 then
      return self:deep(n - 1)
    end
    return self.value
  end

  local obj = setmetatable({ value = 202 }, { __index = proto })
  assert(obj:deep(30000) == 202)
end

-- Tail call forwarding varargs
do
  local a = {}
  function a:id(...)
    return ...
  end
  function a:forward(...)
    return self:id(...)
  end

  assert(select("#", a:forward(1, nil, "x")) == 3)
  local x, y, z = a:forward(1, nil, "x")
  assert(x == 1 and y == nil and z == "x")
end

-- Non-method tail calls (sanity check)
do
  local function f(n) if n > 0 then return f(n - 1) else return 42 end end
  assert(f(50000) == 42)
end

-- Tail call through __call metamethod
do
  local mt = {__call = function(self, n)
    if n > 0 then return self(n - 1) else return 77 end
  end}
  local t = setmetatable({}, mt)
  assert(t(1000) == 77)
end

-- Method tail call returning multiple values
do
  local b = {}
  function b:multi(n)
    if n > 0 then return self:multi(n - 1)
    else return 1, 2, 3 end
  end
  local x, y, z = b:multi(10000)
  assert(x == 1 and y == 2 and z == 3)
end

-- Method tail call preserves self identity
do
  local c = {}
  function c:identity(n)
    if n > 0 then return self:identity(n - 1) else return self end
  end
  assert(c:identity(1000) == c)
end

-- Accumulator pattern (tail position despite computation)
do
  local d = {}
  function d:sum(n, acc)
    acc = acc or 0
    if n > 0 then
      return self:sum(n - 1, acc + n)
    else
      return acc
    end
  end
  assert(d:sum(100) == 5050)
end

-- Mutual method tail calls
do
  local e = {}
  function e:even(n)
    if n == 0 then return true end
    return self:odd(n - 1)
  end
  function e:odd(n)
    if n == 0 then return false end
    return self:even(n - 1)
  end
  assert(e:even(10000) == true)
  assert(e:odd(10001) == true)
end

print("OK")
