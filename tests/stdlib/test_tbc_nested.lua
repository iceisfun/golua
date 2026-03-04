-- Test: To-be-closed inside close methods (nested tbc)
-- From: locals.lua
-- What: Tests that to-be-closed variables can be used inside __close metamethods, including proper ordering and error propagation in nested close scenarios.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  -- tbc inside close methods
  local track = {}
  local function foo ()
    local x <close> = func2close(function ()
      local xx <close> = func2close(function (_, msg)
        assert(msg == nil)
        track[#track + 1] = "xx"
      end)
      track[#track + 1] = "x"
    end)
    track[#track + 1] = "foo"
    return 20, 30, 40
  end
  local a, b, c, d = foo()
  assert(a == 20 and b == 30 and c == 40 and d == nil)
  assert(track[1] == "foo" and track[2] == "x" and track[3] == "xx")

  -- again, with errors
  local track = {}
  local function foo ()
    local x0 <close> = func2close(function (_, msg)
      assert(msg == 202)
        track[#track + 1] = "x0"
    end)
    local x <close> = func2close(function ()
      local xx <close> = func2close(function (_, msg)
        assert(msg == 101)
        track[#track + 1] = "xx"
        error(202)
      end)
      track[#track + 1] = "x"
      error(101)
    end)
    track[#track + 1] = "foo"
    return 20, 30, 40
  end
  local st, msg = pcall(foo)
  assert(not st and msg == 202)
  assert(track[1] == "foo" and track[2] == "x" and track[3] == "xx" and
         track[4] == "x0")
end
