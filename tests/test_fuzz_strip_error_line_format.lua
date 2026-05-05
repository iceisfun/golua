-- broken_fuzz_strip_error_line_format:
-- Errors raised inside a stripped (string.dump(f, true)) function report
-- the location as "?:-1:" (Lua 5.4 format) instead of "?:?:" (Lua 5.5 format).
--
-- BROKEN: vm/vm_error.go around lines 21 and 59 hard-codes "%s:-1: %s"
-- for stripped/missing-line error formatting. The comment notes "matches
-- Lua 5.4" — true, but Lua 5.5 changed the format to print "?" for an
-- unknown line. golua kept the 5.4 format on the lua_5_5_0 branch.
--
-- Reference (lua5.5.0):
--   ?:?: attempt to index a nil value
--
-- Reference (lua 5.4.8):
--   ?:-1: attempt to index a nil value     -- 5.4 still uses -1
--
-- golua today (lua_5_5_0 branch):
--   ?:-1: attempt to index a nil value     -- still 5.4 format
--
-- Discovered: differential fuzz 2026-05-04 (load wave-1 agent).
-- Parity gap, low priority. Fix: change "%s:-1: %s" to "%s:?: %s" in
-- vm/vm_error.go for the lua_5_5_0 branch.

local f = load("local x = nil; return x.y", "@orig.lua")
local stripped = load(string.dump(f, true))
local ok, err = pcall(stripped)

assert(ok == false, "stripped function should error")
-- 5.5 prefix is "?:?:" not "?:-1:"
assert(err:match("^%?:%?:"),
  "expected stripped error to start with '?:?:' (Lua 5.5 format); got: " .. err)

print("ok")
