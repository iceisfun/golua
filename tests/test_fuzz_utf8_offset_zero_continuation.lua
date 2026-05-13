-- test_fuzz_utf8_offset_zero_continuation:
-- utf8.offset(s, 0, i) must verify that the resolved start byte is not a
-- continuation byte. Lua 5.5 added this check (lutf8lib.c:217-218); Lua
-- 5.4 returned the position without checking. golua matches the 5.5
-- behavior on this branch.
--
-- For s = "\xB0+" (orphan continuation, then ASCII '+'),
-- `utf8.offset(s, 0, 1)` must raise "initial position is a continuation
-- byte".
--
-- Reference (lua5.5.0):
--   pcall(utf8.offset, "\xB0+", 0, 1)
--     -> false, "initial position is a continuation byte"
--
-- Reference (lua 5.4.8):
--   pcall(utf8.offset, "\xB0+", 0, 1) -> true, 1   (no check in 5.4)
--
-- Discovered: differential fuzz 2026-05-04 (utf8 wave-2 agent) as a
-- 5.4→5.5 parity gap; fixed in stdlib/utf8.go (n==0 rewind path).

local s = string.char(0xB0, 0x2B)  -- continuation byte, '+'

local ok, err = pcall(utf8.offset, s, 0, 1)
assert(ok == false,
  "utf8.offset(s, 0, 1) should error when start byte is a continuation byte (5.5)")
assert(type(err) == "string" and err:find("continuation byte"),
  "expected 'continuation byte' in error; got " .. tostring(err))

print("ok")
