-- Test that debug.getinfo(1).namewhat returns "hook" consistently
-- when called from inside a hook function, regardless of what
-- bytecode instruction triggered the hook.
local debug = require "debug"
local results = {}
local function f()
  results[#results+1] = debug.getinfo(1).namewhat
end
debug.sethook(f, "l")
local a = 0
_ENV.a = a
a = 1
debug.sethook()

-- All calls should report "hook"
for i, r in ipairs(results) do
  assert(r == "hook", "call " .. i .. " got '" .. r .. "' expected 'hook'")
end
assert(#results >= 3, "expected at least 3 hook calls, got " .. #results)

-- The hook frame stays a hook frame for an xpcall message handler after an
-- error from the hook has unwound it, rather than taking the name of the
-- call its caller is still in.
local P = print
local tb, nw
local n = 0
xpcall(function()
  -- the third line event is the one for the CALL of P, back on its line
  debug.sethook(function()
    n = n + 1
    if n == 3 then
      debug.sethook()
      error("boom")
    end
  end, "l")
  P(
    1)
end, function(m)
  nw = debug.getinfo(3, "n").namewhat
  tb = debug.traceback(m)
end)
assert(nw == "hook", nw)
assert(tb:find("in hook '?'", 1, true), tb)
assert(not tb:find("upvalue 'P'", 1, true), tb)

-- A function the hook tail-calls is not the hook.
local tnw
local function g() tnw = debug.getinfo(1, "n").namewhat end
debug.sethook(function() debug.sethook() return g() end, "l")
a = 2
assert(tnw == "", tnw)
print("PASS") --> PASS
