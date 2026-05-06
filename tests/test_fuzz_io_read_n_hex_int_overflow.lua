-- broken_fuzz_io_read_n_hex_int_overflow:
-- f:read('n') on a hex integer literal that exceeds uint64 returns nil
-- instead of wrapping mod 2^64 (the documented golua/Lua 5.5 behavior
-- for tonumber/lexer).
--
-- BROKEN: stdlib/io.go around line 1108 uses
--   strconv.ParseUint(hexBody, 16, 64)
-- which rejects values > 2^64. The lexer and tonumber both wrap mod 2^64
-- (see tests/doctest/hex_overflow.lua). f:read('n') should match.
--
-- Reference (lua5.5.0):
--   tonumber('0x10000000000000000') -> 0   (wraps mod 2^64)
--   write "0x10000000000000000" to file; f:read('n') -> 0   (same wrap)
--
-- Reference (lua 5.4.8): same.
--
-- golua today:
--   tonumber wraps: 0   (already correct)
--   f:read('n') -> nil  (wrong)
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("0x10000000000000000"); w:close()
local r = io.open(p, "r")
local got = r:read("n")
r:close()
os.remove(p)

assert(got == 0,
  "f:read('n') on '0x1' .. '0' x 16 should wrap to 0; got " .. tostring(got))

print("ok")
