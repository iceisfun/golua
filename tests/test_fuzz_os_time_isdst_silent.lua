-- broken_fuzz_os_time_isdst_silent:
-- os.time({...isdst=true}) silently succeeds under TZ=UTC (or any zone
-- without DST), instead of erroring like reference Lua.
--
-- BROKEN: vm/default_os.go around lines 148-150 (resolveLocalTime). When
-- the caller requests a DST state that doesn't exist for the active zone,
-- C's mktime sets tm_isdst=-1 effectively / returns -1; reference Lua
-- propagates that as "time result cannot be represented in this
-- installation". golua falls back silently to the base (non-DST) result.
--
-- Same root cause makes any TRUTHY value for isdst (numbers, strings)
-- succeed instead of erroring or being normalized; only nil/false should
-- mean "no DST hint".
--
-- Reference (lua5.5.0 and lua 5.4.8 — same), under TZ=UTC LC_ALL=C:
--   pcall(os.time, {year=2000, month=1, day=1, hour=0, min=0, sec=0, isdst=true})
--     -> false, "time result cannot be represented in this installation"
--
-- golua today:
--   -> true, 946684800   (silently produces a result that doesn't reflect the request)
--
-- Discovered: differential fuzz 2026-05-04 (os wave-3 agent).
-- Run this test under TZ=UTC LC_ALL=C for stable behavior.

-- This test only makes sense in a no-DST zone; in a DST zone the request
-- can be satisfied so the call legitimately succeeds. Detect DST-awareness
-- at runtime by sampling July (northern-hemisphere DST) and skipping if so.
-- Honour TZ=UTC explicitly to allow forced UTC runs.
local tz = os.getenv("TZ") or ""
if tz ~= "UTC" then
  local sample = os.date("*t", os.time{year=2000, month=7, day=1, hour=12})
  if sample and sample.isdst then
    print("skip: requires no-DST zone (TZ=UTC)")
    return
  end
end

local ok, err = pcall(os.time,
  {year=2000, month=1, day=1, hour=0, min=0, sec=0, isdst=true})
assert(ok == false,
  "os.time with isdst=true in non-DST zone must error; got ok=" .. tostring(ok) ..
  " result=" .. tostring(err))
assert(type(err) == "string" and err:find("cannot be represented"),
  "expected 'cannot be represented' in error; got " .. tostring(err))

print("ok")
