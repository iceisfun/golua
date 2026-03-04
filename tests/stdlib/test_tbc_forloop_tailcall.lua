-- Test: To-be-closed in for loops prevents tail calls (5.4.3 bug)
-- From: locals.lua
-- What: Tests that to-be-closed variables created by for loops also prevent tail calls, fixing a bug from Lua 5.4.3.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  local closed = false

  local function foo ()
    return function () return true end, 0, 0,
           func2close(function () closed = true end)
  end

  local function tail() return closed end

  local function foo1 ()
    for k in foo() do return tail() end
  end

  assert(foo1() == false)
  assert(closed == true)
end
