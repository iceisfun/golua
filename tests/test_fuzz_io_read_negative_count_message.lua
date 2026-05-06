-- broken_fuzz_io_read_negative_count_message:
-- f:read(-N) error message says "not enough memory" instead of reference
-- Lua's "resulting string too large".
--
-- BROKEN: stdlib/io.go around line 986 panics with "not enough memory".
-- Reference C Lua's liolib.c uses "resulting string too large" (LSTR_ERR).
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(f.read, f, -1) -> false, "resulting string too large"
--
-- golua today:
--   -> false, "not enough memory"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("hi"); w:close()
local r = io.open(p, "r")

local ok, err = pcall(r.read, r, -1)
r:close(); os.remove(p)

assert(ok == false, "read with negative count must fail")
assert(type(err) == "string", "error must be a string")
assert(err:find("resulting string too large"),
  "expected 'resulting string too large'; got: " .. err)

print("ok")
