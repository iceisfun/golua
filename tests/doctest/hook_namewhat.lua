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
print("PASS") --> PASS
