-- test_fuzz_strip_error_line_format:
-- Errors raised inside a stripped (string.dump(f, true)) function must
-- report the location as "?:?:" (Lua 5.5 format), not "?:-1:" (Lua 5.4
-- format). Lua 5.5 prints "?" for an unknown line.
--
-- Reference (lua5.5.0):
--   ?:?: attempt to index a nil value
--
-- Reference (lua 5.4.8):
--   ?:-1: attempt to index a nil value     -- 5.4 still uses -1
--
-- Discovered: differential fuzz 2026-05-04 (load wave-1 agent) as a
-- 5.4→5.5 parity gap; fixed in vm/vm_error.go.

local f = load("local x = nil; return x.y", "@orig.lua")
local stripped = load(string.dump(f, true))
local ok, err = pcall(stripped)

assert(ok == false, "stripped function should error")
-- 5.5 prefix is "?:?:" not "?:-1:"
assert(err:match("^%?:%?:"),
  "expected stripped error to start with '?:?:' (Lua 5.5 format); got: " .. err)

print("ok")
