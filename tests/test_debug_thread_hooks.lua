-- Test: debug.sethook/gethook([thread,] ...) with coroutine thread argument
-- Verifies that hooks can be set/queried independently on coroutines.

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
-- Basic: sethook(thread, hook, mask) and gethook(thread)
----------------------------------------------------------------------
do
  local log = {}
  local co = coroutine.create(function(x)
    local y = x + 1
    coroutine.yield(y)
    return y
  end)

  local function hook(event) log[#log+1] = event end
  debug.sethook(co, hook, "cr")

  -- gethook(thread) returns the coroutine's hook
  local h, m, c = debug.gethook(co)
  check("co hook is hook", h == hook, true)
  check("co mask", m, "cr")
  check("co count", c, 0)

  -- Main thread should NOT have that hook
  local mh, mm, mc = debug.gethook()
  check("main hook nil", mh, nil)

  -- Resume: hook fires on coroutine
  coroutine.resume(co, 10)
  check("events fired", #log > 0, true)

  -- Remove hook from coroutine
  debug.sethook(co)
  local h2, m2, c2 = debug.gethook(co)
  check("after remove hook nil", h2, nil)

  -- Resume again: no events (hook removed)
  log = {}
  coroutine.resume(co)
  check("no events after remove", #log, 0)
end

----------------------------------------------------------------------
-- sethook with count on coroutine
----------------------------------------------------------------------
do
  local count_events = 0
  local co = coroutine.create(function()
    local x = 0
    for i = 1, 10 do x = x + i end
    coroutine.yield(x)
    return x
  end)

  debug.sethook(co, function() count_events = count_events + 1 end, "", 1)
  coroutine.resume(co)
  check("count events fired", count_events > 0, true)

  local h, m, c = debug.gethook(co)
  check("count value", c, 1)
end

----------------------------------------------------------------------
-- sethook with nil removes hook from coroutine
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    coroutine.yield()
    return 1
  end)

  debug.sethook(co, function() end, "c")
  local h1 = debug.gethook(co)
  check("hook set before nil", h1 ~= nil, true)

  debug.sethook(co, nil)
  local h2 = debug.gethook(co)
  check("hook nil after sethook nil", h2, nil)
end

----------------------------------------------------------------------
-- Independent hooks: main thread vs coroutine
----------------------------------------------------------------------
do
  local main_log = {}
  local co_log = {}

  local co = coroutine.create(function()
    local x = 1
    coroutine.yield()
    local y = 2
    return y
  end)

  -- Set line hook on main thread
  debug.sethook(function(ev, line) main_log[#main_log+1] = ev end, "l")

  -- Set call hook on coroutine
  debug.sethook(co, function(ev) co_log[#co_log+1] = ev end, "c")

  coroutine.resume(co)

  -- Remove main hook
  debug.sethook()

  check("main had line events", #main_log > 0, true)
  check("co had call events", #co_log > 0, true)

  -- Verify hooks are independent after removal
  local mh = debug.gethook()
  local ch = debug.gethook(co)
  check("main hook removed", mh, nil)
  check("co hook still set", ch ~= nil, true)
end

----------------------------------------------------------------------
-- sethook with all mask types: "crl"
----------------------------------------------------------------------
do
  local events = {}
  local co = coroutine.create(function()
    local x = 1
    local function foo() return x end
    foo()
    coroutine.yield()
  end)

  debug.sethook(co, function(ev, line)
    if ev == "line" then
      events[#events+1] = ev..":"..tostring(line)
    else
      events[#events+1] = ev
    end
  end, "crl")

  coroutine.resume(co)

  check("crl total events", #events > 5, true)

  -- gethook returns full mask
  local h, m, c = debug.gethook(co)
  check("crl mask", m, "crl")
end

----------------------------------------------------------------------
-- sethook on dead coroutine: no error
----------------------------------------------------------------------
do
  local co = coroutine.create(function() end)
  coroutine.resume(co)
  -- co is dead

  local ok, err = pcall(debug.sethook, co, function() end, "c")
  check("sethook on dead ok", ok, true)

  -- gethook on dead coroutine returns the hook
  local h = debug.gethook(co)
  check("gethook on dead", h ~= nil, true)
end

----------------------------------------------------------------------
-- gethook on never-resumed coroutine: nil
----------------------------------------------------------------------
do
  local co = coroutine.create(function() end)
  -- Never resumed

  local h, m, c = debug.gethook(co)
  check("never resumed hook", h, nil)
  check("never resumed mask", m, nil)
  check("never resumed count", c, nil)
end

----------------------------------------------------------------------
-- sethook on suspended coroutine
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    coroutine.yield()
  end)
  coroutine.resume(co)
  -- co is suspended

  debug.sethook(co, function() end, "c")
  check("sethook on suspended ok", true, true)

  local h = debug.gethook(co)
  check("gethook on suspended", h ~= nil, true)
end

----------------------------------------------------------------------
-- Hook set on coroutine sees correct events across yields
----------------------------------------------------------------------
do
  local events = {}
  local co = coroutine.create(function()
    coroutine.yield(1)
    coroutine.yield(2)
    return 3
  end)

  debug.sethook(co, function(ev)
    events[#events+1] = ev
  end, "cr")

  coroutine.resume(co)
  local after_first = #events

  coroutine.resume(co)
  local after_second = #events

  coroutine.resume(co)
  local after_third = #events

  check("events after first yield", after_first > 0, true)
  check("events grow on second", after_second > after_first, true)
  check("events grow on third", after_third > after_second, true)
end

----------------------------------------------------------------------
-- Normal sethook (no thread arg) still works
----------------------------------------------------------------------
do
  local events = {}
  debug.sethook(function(ev) events[#events+1] = ev end, "c")
  local function foo() end
  foo()
  debug.sethook()

  check("normal sethook works", #events > 0, true)
end

----------------------------------------------------------------------
-- Normal gethook (no thread arg) still works
----------------------------------------------------------------------
do
  local function myhook() end
  debug.sethook(myhook, "cr", 5)
  local h, m, c = debug.gethook()
  check("normal gethook func", h == myhook, true)
  check("normal gethook mask", m, "cr")
  check("normal gethook count", c, 5)
  debug.sethook()  -- clean up
end

----------------------------------------------------------------------
-- Hook on coroutine does not affect main thread's getinfo levels
----------------------------------------------------------------------
do
  local co = coroutine.create(function()
    coroutine.yield()
  end)
  coroutine.resume(co)

  debug.sethook(co, function() end, "c")

  -- Main thread getinfo should still work normally
  local info = debug.getinfo(1, "S")
  check("main getinfo unaffected", info.source ~= nil, true)
end

----------------------------------------------------------------------
print(string.format("\nhook thread tests: %d passed, %d failed", pass, fail))
assert(fail == 0, fail .. " tests failed")
