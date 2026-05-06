-- broken_fuzz_io_read_n_hex_float_overflow:
-- f:read('n') on a hex float that overflows to ±inf returns nil instead
-- of inf, breaking parity with reference Lua.
--
-- BROKEN: stdlib/io.go around lines 1080-1091 (parseReadNumber hex-float
-- branch). When strconv.ParseFloat returns (+Inf, ErrRange), the code
-- falls through to parseHexFloat which fails (line 1171 ParseUint can't
-- parse "1p1024"). The ErrRange/Inf fallback at line ~1122 only runs in
-- the non-hex float path.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   write "0x1p1024" to a file, f:read('n') returns inf
--
-- golua today: returns nil
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("0x1p1024"); w:close()
local r = io.open(p, "r")
local got = r:read("n")
r:close()
os.remove(p)

assert(got == math.huge,
  "f:read('n') on '0x1p1024' should be inf; got " .. tostring(got))

print("ok")
