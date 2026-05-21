-- test_fuzz_for_float_limit_2pow63:
-- A numeric `for` with an integer init/step and a float limit exactly equal
-- to 2^63 (9223372036854775808.0) must clamp the limit to math.maxinteger,
-- not silently skip (positive step) or wrongly run forever (negative step).
--
-- BROKEN (before fix): vm/vm_exec.go's integer-for float-limit path used
--   fl > float64(math.MaxInt64)
-- but float64(math.MaxInt64) itself rounds up to 2^63, so `2^63 > 2^63` is
-- false. The code fell through to int64(fl), and int64(2^63) overflows in Go
-- to math.MinInt64 — making a positive-step loop skip entirely and a
-- negative-step loop run when it should not. Fix: use `>=`.
--
-- Reference (lua5.5.0):
--   for i=1,2^63    do .. end  -> runs (limit clamps to maxinteger)
--   for i=1,2^63,-1 do .. end  -> never runs (+2^63 is effectively +inf)
--
-- Discovered: differential scout 2026-05-20 (control-flow agent).

local function count(lo, hi, st)
  local c = 0
  for _ = lo, hi, st do
    c = c + 1
    if c > 5 then break end
  end
  return c
end

-- positive step, limit exactly 2^63: loop must run
assert(count(1, 2^63, 1) == 6, "for 1,2^63 (step 1) must run")
-- negative step, limit exactly +2^63: loop must NOT run
assert(count(1, 2^63, -1) == 0, "for 1,2^63 (step -1) must not run")

-- neighbouring values must keep working
assert(count(1, 9.2e18, 1) == 6, "for 1,9.2e18 must run")
assert(count(1, 2^64, 1) == 6, "for 1,2^64 must run")
assert(count(1, 2^64, -1) == 0, "for 1,2^64 (step -1) must not run")
assert(count(1, -(2^63), -1) == 6, "for 1,-2^63 (step -1) must run")
assert(count(1, math.huge, 1) == 6, "for 1,math.huge must run")

print("ok")
