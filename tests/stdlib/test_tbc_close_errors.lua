-- Test: Errors in __close metamethods
-- From: locals.lua
-- What: Tests error propagation through multiple __close metamethods: errors chain through close methods, and each close method receives the current error as its second argument.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end


  -- original error is in __close
  local function foo ()

    local x <close> =
      func2close(function (self, msg)
        assert(string.find(msg, "@y"))
        error("@x")
      end)

    local x1 <close> =
      func2close(function (self, msg)
        assert(string.find(msg, "@y"))
      end)

    local gc <close> = func2close(function () collectgarbage() end)

    local y <close> =
      func2close(function (self, msg)
        assert(string.find(msg, "@z"))  -- error in 'z'
        error("@y")
      end)

    local z <close> =
      func2close(function (self, msg)
        assert(msg == nil)
        error("@z")
      end)

    return 200
  end

  local stat, msg = pcall(foo, false)
  assert(string.find(msg, "@x"))


  -- original error not in __close
  local function foo ()

    local x <close> =
      func2close(function (self, msg)
        -- after error, 'foo' was discarded, so caller now
        -- must be 'pcall'
        assert(debug.getinfo(2).name == "pcall")
        assert(string.find(msg, "@x1"))
      end)

    local x1 <close> =
      func2close(function (self, msg)
        assert(debug.getinfo(2).name == "pcall")
        assert(string.find(msg, "@y"))
        error("@x1")
      end)

    local gc <close> = func2close(function () collectgarbage() end)

    local y <close> =
      func2close(function (self, msg)
        assert(debug.getinfo(2).name == "pcall")
        assert(string.find(msg, "@z"))
        error("@y")
      end)

    local first = true
    local z <close> =
      func2close(function (self, msg)
        assert(debug.getinfo(2).name == "pcall")
        -- 'z' close is called once
        assert(first and msg == 4)
        first = false
        error("@z")
      end)

    error(4)    -- original error
  end

  local stat, msg = pcall(foo, true)
  assert(string.find(msg, "@x1"))

  -- error leaving a block
  local function foo (...)
    do
      local x1 <close> =
        func2close(function (self, msg)
          assert(string.find(msg, "@X"))
          error("@Y")
        end)

      local x123 <close> =
        func2close(function (_, msg)
          assert(msg == nil)
          error("@X")
        end)
    end
    os.exit(false)    -- should not run
  end

  local st, msg = xpcall(foo, debug.traceback)
  assert(string.match(msg, "^[^ ]* @Y"))

  -- error in toclose in vararg function
  local function foo (...)
    local x123 <close> = func2close(function () error("@x123") end)
  end

  local st, msg = xpcall(foo, debug.traceback)
  assert(string.match(msg, "^[^ ]* @x123"))
  assert(string.find(msg, "in metamethod 'close'"))
end
