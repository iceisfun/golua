-- Test: __close vs return hooks in Lua functions
-- From: locals.lua
-- What: Tests the interaction between to-be-closed variable __close metamethods and debug return hooks in Lua functions, verifying the correct ordering of close calls and hook events.

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

  -- '__close' vs. return hooks in Lua functions
  local trace = {}

  local function hook (event)
    trace[#trace + 1] = event .. " " .. debug.getinfo(2).name
  end

  local function foo (...)
    local x <close> = func2close(function (_,msg)
      trace[#trace + 1] = "x"
    end)

    local y <close> = func2close(function (_,msg)
      debug.sethook(hook, "r")
    end)

    return ...
  end

  local t = {foo(10,20,30)}
  debug.sethook()
  checktable(t, {10, 20, 30})
  checktable(trace,
    {"return sethook", "return close", "x", "return close", "return foo"})
end
