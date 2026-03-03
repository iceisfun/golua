-- Test: Yielding inside closing metamethods after an error
-- From: locals.lua
-- What: Tests yielding inside __close metamethods when errors have occurred, verifying that error messages are properly threaded through multiple close/yield cycles within pcall.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  local co = coroutine.wrap(function ()

    local function foo (err)

      local z <close> = func2close(function(_, msg)
        assert(msg == nil or msg == err + 20)
        coroutine.yield("z")
        return 100, 200
      end)

      local y <close> = func2close(function(_, msg)
        -- still gets the original error (if any)
        assert(msg == err or (msg == nil and err == 1))
        coroutine.yield("y")
        if err then error(err + 20) end   -- creates or changes the error
      end)

      local x <close> = func2close(function(_, msg)
        assert(msg == err or (msg == nil and err == 1))
        coroutine.yield("x")
        return 100, 200
      end)

      if err == 10 then error(err) else return 10, 20 end
    end

    coroutine.yield(pcall(foo, nil))  -- no error
    coroutine.yield(pcall(foo, 1))    -- error in __close
    return pcall(foo, 10)     -- 'foo' will raise an error
  end)

  local a, b = co()   -- first foo: no error
  assert(a == "x" and b == nil)    -- yields inside 'x'; Ok
  a, b = co()
  assert(a == "y" and b == nil)    -- yields inside 'y'; Ok
  a, b = co()
  assert(a == "z" and b == nil)    -- yields inside 'z'; Ok
  local a, b, c = co()
  assert(a and b == 10 and c == 20)   -- returns from 'pcall(foo, nil)'

  local a, b = co()   -- second foo: error in __close
  assert(a == "x" and b == nil)    -- yields inside 'x'; Ok
  a, b = co()
  assert(a == "y" and b == nil)    -- yields inside 'y'; Ok
  a, b = co()
  assert(a == "z" and b == nil)    -- yields inside 'z'; Ok
  local st, msg = co()             -- reports the error in 'y'
  assert(not st and msg == 21)

  local a, b = co()    -- third foo: error in function body
  assert(a == "x" and b == nil)    -- yields inside 'x'; Ok
  a, b = co()
  assert(a == "y" and b == nil)    -- yields inside 'y'; Ok
  a, b = co()
  assert(a == "z" and b == nil)    -- yields inside 'z'; Ok
  local st, msg = co()    -- gets final error
  assert(not st and msg == 10 + 20)

end
