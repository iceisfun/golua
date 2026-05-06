-- broken_fuzz_io_seek_setvbuf_float_error:
-- f:seek('set', non-integer-float) and f:setvbuf('full', non-integer-float)
-- report "number expected" instead of reference Lua's "number has no
-- integer representation".
--
-- BROKEN: stdlib/io.go around line 1340 (seek) and ~1379 (setvbuf): when
-- ToInt() fails, raise the wrong error class. Should distinguish "got
-- non-number" vs "got non-integer-representable number".
--
-- Affects 2.5, 1.5, NaN, ±Inf, and any float without an integer
-- representation.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(f.seek, f, 'set', 2.5)
--     -> false, "bad argument #3 to '?' (number has no integer representation)"
--   pcall(w.setvbuf, w, 'full', 1.5)
--     -> false, ".. (number has no integer representation)"
--
-- golua today:
--   -> "(number expected)"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
local w = io.open(p, "w"); w:write("hello"); w:close()
local r = io.open(p, "r")

for _, off in ipairs({2.5, 1.5, math.huge, -math.huge}) do
  local ok, err = pcall(r.seek, r, "set", off)
  assert(ok == false, "seek with non-integer float must fail")
  assert(err:find("no integer representation"),
    "seek with " .. tostring(off) .. ": expected 'no integer representation'; got: " .. err)
end

local ww = io.open(p, "w")
local ok, err = pcall(ww.setvbuf, ww, "full", 1.5)
ww:close(); r:close(); os.remove(p)
assert(ok == false, "setvbuf with non-integer float must fail")
assert(err:find("no integer representation"),
  "setvbuf: expected 'no integer representation'; got: " .. err)

print("ok")
