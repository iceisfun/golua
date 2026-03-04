-- Test: To-be-closed yielding in coroutines
-- From: locals.lua
-- What: Tests that __close metamethods can yield inside coroutines, and that the close ordering and pcall interactions work correctly across yields.

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

  -- yielding inside closing metamethods

  local trace = {}
  local co = coroutine.wrap(function ()

    trace[#trace + 1] = "nowX"

    -- will be closed after 'y'
    local x <close> = func2close(function (_, msg)
      assert(msg == nil)
      trace[#trace + 1] = "x1"
      coroutine.yield("x")
      trace[#trace + 1] = "x2"
    end)

    return pcall(function ()
      do   -- 'z' will be closed first
        local z <close> = func2close(function (_, msg)
          assert(msg == nil)
          trace[#trace + 1] = "z1"
          coroutine.yield("z")
          trace[#trace + 1] = "z2"
        end)
      end

      trace[#trace + 1] = "nowY"

      -- will be closed after 'z'
      local y <close> = func2close(function(_, msg)
        assert(msg == nil)
        trace[#trace + 1] = "y1"
        coroutine.yield("y")
        trace[#trace + 1] = "y2"
      end)

      return 10, 20, 30
    end)
  end)

  assert(co() == "z")
  assert(co() == "y")
  assert(co() == "x")
  checktable({co()}, {true, 10, 20, 30})
  checktable(trace, {"nowX", "z1", "z2", "nowY", "y1", "y2", "x1", "x2"})

end
