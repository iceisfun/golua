-- Test: debug.getinfo([thread,] f [, what]) with coroutine thread argument
-- Verifies that getinfo accepts a coroutine as first arg and inspects
-- that coroutine's call stack rather than the current thread's.

local pass, fail = 0, 0
local function check(name, got, expect)
  if got == expect then
    pass = pass + 1
  else
    fail = fail + 1
    print("FAIL " .. name .. ": got " .. tostring(got) .. ", expected " .. tostring(expect))
  end
end

----------------------------------------------------------------------
-- Basic: getinfo(thread, level, what) on a suspended coroutine
----------------------------------------------------------------------
do
  local co = coroutine.create(function(a, b)
    local c = a + b
    coroutine.yield(c)
    return c
  end)
  coroutine.resume(co, 3, 4)

  -- Level 0 = yield itself (C function)
  local i0 = debug.getinfo(co, 0, "Slntu")
  check("L0 what", i0.what, "C")
  check("L0 name", i0.name, "yield")
  check("L0 currentline", i0.currentline, -1)
  check("L0 istailcall", i0.istailcall, false)

  -- Level 1 = the coroutine body (Lua function)
  local i1 = debug.getinfo(co, 1, "Slntu")
  check("L1 what", i1.what, "Lua")
  check("L1 nparams", i1.nparams, 2)
  check("L1 isvararg", i1.isvararg, false)
  check("L1 currentline type", type(i1.currentline), "number")
  check("L1 linedefined type", type(i1.linedefined), "number")

  -- Level 2 = out of range, returns nil
  local i2 = debug.getinfo(co, 2)
  check("L2 out of range", i2, nil)

  -- Level 99 = way out of range
  local i99 = debug.getinfo(co, 99)
  check("L99 out of range", i99, nil)
end

----------------------------------------------------------------------
-- Default what string (no what argument)
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    coroutine.yield()
  end)
  coroutine.resume(co)

  local info = debug.getinfo(co, 1)
  check("default has source", info.source ~= nil, true)
  check("default has what", info.what ~= nil, true)
  check("default has func", info.func ~= nil, true)
  check("default has nups type", type(info.nups), "number")
end

----------------------------------------------------------------------
-- "f" flag returns the function value
----------------------------------------------------------------------
do
  local function target()
    coroutine.yield()
  end
  local co = coroutine.create(target)
  coroutine.resume(co)

  local info = debug.getinfo(co, 1, "f")
  check("f flag func type", type(info.func), "function")
end

----------------------------------------------------------------------
-- "t" flag (istailcall)
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    coroutine.yield()
  end)
  coroutine.resume(co)

  local i0 = debug.getinfo(co, 0, "t")
  check("t flag L0 istailcall", i0.istailcall, false)
end

----------------------------------------------------------------------
-- "L" flag (activelines)
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    local x = 1
    local y = 2
    coroutine.yield()
  end)
  coroutine.resume(co)

  local info = debug.getinfo(co, 1, "L")
  check("L flag has activelines", info.activelines ~= nil, true)
end

----------------------------------------------------------------------
-- getinfo(thread, func, what) - function form with thread
----------------------------------------------------------------------
do
  local function myfunc(a, b, c) end
  local co = coroutine.create(function() coroutine.yield() end)
  coroutine.resume(co)

  local info = debug.getinfo(co, myfunc, "u")
  check("func form nparams", info.nparams, 3)
  check("func form isvararg", info.isvararg, false)
end

----------------------------------------------------------------------
-- Dead coroutine: getinfo returns nil
----------------------------------------------------------------------
do
  local co = coroutine.create(function() return 1 end)
  coroutine.resume(co)
  -- co is now dead

  local info = debug.getinfo(co, 0)
  check("dead co level 0", info, nil)

  local info1 = debug.getinfo(co, 1)
  check("dead co level 1", info1, nil)
end

----------------------------------------------------------------------
-- Never-resumed coroutine: getinfo returns nil
----------------------------------------------------------------------
do
  local co = coroutine.create(function(x) return x end)
  -- Never resumed

  local info = debug.getinfo(co, 0)
  check("never resumed L0", info, nil)
end

----------------------------------------------------------------------
-- Nested coroutines: getinfo on inner vs outer
----------------------------------------------------------------------
do
  local co_inner
  local co_outer = coroutine.create(function()
    co_inner = coroutine.create(function()
      coroutine.yield()
    end)
    coroutine.resume(co_inner)
    coroutine.yield()
  end)
  coroutine.resume(co_outer)

  -- Both are suspended
  local i_outer = debug.getinfo(co_outer, 1, "S")
  local i_inner = debug.getinfo(co_inner, 1, "S")
  check("outer what", i_outer.what, "Lua")
  check("inner what", i_inner.what, "Lua")
end

----------------------------------------------------------------------
-- Vararg function in coroutine
----------------------------------------------------------------------
do
  local co = coroutine.create(function(...)
    coroutine.yield()
  end)
  coroutine.resume(co, 1, 2, 3)

  local info = debug.getinfo(co, 1, "u")
  check("vararg isvararg", info.isvararg, true)
end

----------------------------------------------------------------------
-- Multiple yield levels: deeper stack in coroutine
----------------------------------------------------------------------
do
  local function inner()
    coroutine.yield()
  end
  local function middle()
    inner()
  end
  local co = coroutine.create(function()
    middle()
  end)
  coroutine.resume(co)

  -- Stack: L0=yield(C), L1=inner(Lua), L2=middle(Lua), L3=body(Lua)
  local i0 = debug.getinfo(co, 0, "Sn")
  check("deep L0 what", i0.what, "C")

  local i1 = debug.getinfo(co, 1, "Sn")
  check("deep L1 what", i1.what, "Lua")

  local i2 = debug.getinfo(co, 2, "Sn")
  check("deep L2 what", i2.what, "Lua")

  local i3 = debug.getinfo(co, 3, "Sn")
  check("deep L3 what", i3.what, "Lua")

  local i4 = debug.getinfo(co, 4)
  check("deep L4 nil", i4, nil)
end

----------------------------------------------------------------------
-- Non-thread first arg still works normally (number)
----------------------------------------------------------------------
do
  -- getinfo(1) on current thread should still work
  local info = debug.getinfo(1, "S")
  check("normal level 1 has source", info.source ~= nil, true)
end

----------------------------------------------------------------------
-- Non-thread first arg still works normally (function)
----------------------------------------------------------------------
do
  local function f(a, b) end
  local info = debug.getinfo(f, "u")
  check("normal func nparams", info.nparams, 2)
end

----------------------------------------------------------------------
-- Error on invalid first arg (not thread, not number, not function)
----------------------------------------------------------------------
do
  local ok, err = pcall(debug.getinfo, "not a thread", 0)
  check("string arg errors", ok, false)
  check("string arg msg", err:find("number expected") ~= nil, true)
end

----------------------------------------------------------------------
print(string.format("\ngetinfo thread tests: %d passed, %d failed", pass, fail))
assert(fail == 0, fail .. " tests failed")
