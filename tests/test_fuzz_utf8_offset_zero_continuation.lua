-- broken_fuzz_utf8_offset_zero_continuation:
-- utf8.offset(s, 0, i) does not verify that the resolved start byte is not
-- a continuation byte. Lua 5.5 added this check (lutf8lib.c:217-218); golua
-- on the lua_5_5_0 branch still uses 5.4 behavior.
--
-- BROKEN: For s = "\xB0+" (orphan continuation, then ASCII '+'),
-- `utf8.offset(s, 0, 1)` should raise "initial position is a continuation
-- byte" in 5.5. golua returns (1, 1) instead.
--
-- This is a 5.4→5.5 parity gap, NOT a regression vs 5.4. Lua 5.4 returns
-- a single value (1) without checking. The fix is 5.5-specific: golua's
-- lua_5_5_0 branch should match 5.5.
--
-- Reference (lua5.5.0):
--   pcall(utf8.offset, "\xB0+", 0, 1)
--     -> false, "initial position is a continuation byte"
--
-- Reference (lua 5.4.8):
--   pcall(utf8.offset, "\xB0+", 0, 1) -> true, 1   (no check in 5.4)
--
-- golua today (lua_5_5_0 branch):
--   pcall(utf8.offset, "\xB0+", 0, 1) -> true, 1, 1   (still 5.4 behavior)
--
-- Discovered: differential fuzz 2026-05-04 (utf8 wave-2 agent).
-- Suspect: stdlib/utf8.go around lines 391-397 (n==0 rewind path).

local s = string.char(0xB0, 0x2B)  -- continuation byte, '+'

local ok, err = pcall(utf8.offset, s, 0, 1)
assert(ok == false,
  "utf8.offset(s, 0, 1) should error when start byte is a continuation byte (5.5)")
assert(type(err) == "string" and err:find("continuation byte"),
  "expected 'continuation byte' in error; got " .. tostring(err))

print("ok")
