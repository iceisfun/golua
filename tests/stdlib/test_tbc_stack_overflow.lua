-- Test: To-be-closed variable high in the stack (stack overflow)
-- From: locals.lua
-- What: Tests that to-be-closed variables work correctly even when created during stack overflow error handling, high in the stack within a coroutine.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  local function stack(n) n = ((n == 0) or stack(n - 1)) end

  local function checktable (t1, t2)
    assert(#t1 == #t2)
    for i = 1, #t1 do
      assert(t1[i] == t2[i])
    end
  end

  -- test for tbc variable high in the stack

  -- function to force a stack overflow
  local function overflow (n)
    overflow(n + 1)
  end

  -- error handler will create tbc variable handling a stack overflow,
  -- high in the stack
  local function errorh (m)
    assert(string.find(m, "stack overflow"))
    local x <close> = func2close(function (o) o[1] = 10 end)
    return x
  end

  local flag
  local st, obj
  -- run test in a coroutine so as not to swell the main stack
  local co = coroutine.wrap(function ()
    -- tbc variable down the stack
    local y <close> = func2close(function (obj, msg)
      assert(msg == nil)
      obj[1] = 100
      flag = obj
    end)
    st, obj = xpcall(overflow, errorh, 0)
  end)
  co()
  assert(not st and obj[1] == 10 and flag[1] == 100)
end
