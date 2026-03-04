-- Test: To-be-closed does not corrupt return values
-- From: locals.lua
-- What: Tests that closing functions do not corrupt the return values of the enclosing function, and that __close is called before the function returns.

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

  local X = false

  local x, closescope = func2close(function (_, msg)
    stack(10);
    assert(msg == nil)
    X = true
  end, 100)
  assert(x == 100);  x = 101;   -- 'x' is not read-only

  -- closing functions do not corrupt returning values
  local function foo (x)
    local _ <close> = closescope
    return x, X, 23
  end

  local a, b, c = foo(1.5)
  assert(a == 1.5 and b == false and c == 23 and X == true)

  X = false
  foo = function (x)
    local _<close> = func2close(function (_, msg)
      -- without errors, enclosing function should be still active when
      -- __close is called
      assert(debug.getinfo(2).name == "foo")
      assert(msg == nil)
    end)
    local  _<close> = closescope
    local y = 15
    return y
  end

  assert(foo() == 15 and X == true)

  X = false
  foo = function ()
    local x <close> = closescope
    return x
  end

  assert(foo() == closescope and X == true)
end
