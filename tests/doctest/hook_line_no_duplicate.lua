-- Test that line hooks don't fire duplicate events for the same line
-- when a native function call (like debug.getinfo) happens in between.
-- Previously, fireCallHook/fireReturnHook reset lastHookLine to -1,
-- causing the line hook to fire again when returning to the same line.
local debug = require "debug"

local co = coroutine.create(function(x)
  local a = 1
  coroutine.yield(debug.getinfo(1, "l"))
  return a
end)

local tr = {}
local foo = function(e, l) if l then table.insert(tr, l) end end
debug.sethook(co, foo, "lcr")

local _, l = coroutine.resume(co, 10)
-- Should be exactly 2 line events: line 4 (local a = 1) and line 5 (yield)
assert(#tr == 2, "expected 2 line events, got " .. #tr)
assert(tr[1] == l.currentline - 1)
assert(tr[2] == l.currentline)
print("PASS") --> PASS
