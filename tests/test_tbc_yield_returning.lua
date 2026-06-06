-- Test: Yielding inside closing metamethods while returning (5.4.3 bug)
-- From: locals.lua
-- What: Tests yielding inside __close metamethods during function return, including extra yields and table.unpack interactions. Fixes a bug from Lua 5.4.3.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  local function checktable (t1, t2)
    assert(#t1 == #t2)
    for i = 1, #t1 do
      assert(t1[i] == t2[i])
    end
  end

  local extrares    -- result from extra yield (if any)

  local function check (body, extra, ...)
    local t = table.pack(...)   -- expected returns
    local co = coroutine.wrap(body)
    if extra then
      extrares = co()    -- runs until first (extra) yield
    end
    local res = table.pack(co())   -- runs until yield inside '__close'
    -- The "regular" yield yields all values passed to the close function;
    -- without errors, that is only the object being closed. Lua 5.5 passes a
    -- single argument to __close on a normal (error-free) close.
    assert(res.n == 1 and type(res[1]) == "table")
    local res2 = table.pack(co())   -- runs until end of function
    assert(res2.n == t.n)
    for i = 1, #t do
      if t[i] == "x" then
        assert(res2[i] == res[1])    -- value that was closed
      else
        assert(res2[i] == t[i])
      end
    end
  end

  local function foo ()
    local x <close> = func2close(coroutine.yield)
    local extra <close> = func2close(function (self)
      assert(self == extrares)
      coroutine.yield(100)
    end)
    extrares = extra
    return table.unpack{10, x, 30}
  end
  check(foo, true, 10, "x", 30)
  assert(extrares == 100)

  local function foo ()
    local x <close> = func2close(coroutine.yield)
    return
  end
  check(foo, false)

  local function foo ()
    local x <close> = func2close(coroutine.yield)
    local y, z = 20, 30
    return x
  end
  check(foo, false, "x")

  local function foo ()
    local x <close> = func2close(coroutine.yield)
    local extra <close> = func2close(coroutine.yield)
    return table.unpack({}, 1, 100)   -- 100 nils
  end
  check(foo, true, table.unpack({}, 1, 100))

end
