-- broken_fuzz_debug_getlocal_tailcall_temp:
-- debug.getlocal at slot 1 of a Lua frame whose entire body is a tail
-- call (e.g. `return f(...)`) returns "(temporary)" and the called
-- function value. Reference Lua returns nil.
--
-- BROKEN: vm/vm_debug.go around lines 870-953 (GetLocal, frameStackLimit).
-- The frameStackLimit path for "next frame is native" returns
-- nextFrame.base, but for a TAILCALL→native scenario the native frame
-- sits ABOVE the caller's call register, so nextFrame.base exposes
-- the caller's call-temp.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   local f; f = function() return debug.getlocal(1, 1) end
--   print(f())  -> nil
--
-- golua today:
--   -> "(temporary)"  function: 0x...   (leaks the call-register temp)
--
-- Discovered: differential fuzz 2026-05-04 (debug wave-3 agent).

local f
f = function()
  return debug.getlocal(1, 1)
end

local n, v = f()
assert(n == nil,
  "tail-call frame slot 1 should be nil; got name='" .. tostring(n) ..
  "' value=" .. tostring(v))

print("ok")
