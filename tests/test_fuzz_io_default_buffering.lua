-- broken_fuzz_io_default_buffering:
-- File handles opened for writing default to UNBUFFERED mode in golua,
-- whereas reference Lua (and C fopen) defaults to _IOFBF (full buffering).
--
-- BROKEN: vm/full_io.go around line 235 checks
--   bufMode == "no" || bufMode == ""
-- treating empty as no-buffering. Reference C fopen defaults to _IOFBF,
-- so writes are buffered until explicit flush, fclose, or buffer fill.
-- golua should default bufMode to "full" for regular files.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   local w = io.open(p, 'w'); w:write('hello')   -- not flushed yet
--   local r = io.open(p, 'r'); print(r:read('a')) -- "" (buffered, not visible)
--   w:close()
--
-- golua today:
--   The 'r:read' immediately sees "hello" because writes are unbuffered.
--
-- Note: this changes observable behavior — programs that intentionally
-- relied on unbuffered semantics would see a difference. The reference
-- behavior IS the standard, so the fix aligns golua with Lua spec.
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w")
w:write("hello")
-- Do NOT flush or close yet.

local r = io.open(p, "r")
local got = r:read("a")
r:close()

w:close()  -- now flush
os.remove(p)

assert(got == "",
  "default buffering should hide unflushed writes; got '" .. tostring(got) .. "'")

print("ok")
