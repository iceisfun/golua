-- Test os.time isdst field handling
-- This test checks basic isdst functionality without timezone dependency.
-- The Go-level test in vm/os_provider_test.go covers TZ-specific behavior.

-- When isdst=true and isdst=false are both provided for the same wall clock,
-- the difference should be 0 or 3600 (depending on whether the timezone has DST).
local t1 = os.time({year=2000, month=7, day=1, hour=12, min=0, sec=0, isdst=true})
local t2 = os.time({year=2000, month=7, day=1, hour=12, min=0, sec=0, isdst=false})
local diff = t2 - t1

-- In a timezone with DST, diff should be 3600. In UTC or non-DST zones, diff should be 0.
assert(diff == 0 or diff == 3600, "expected 0 or 3600 second difference, got " .. tostring(diff))

-- Without isdst hint, os.time should auto-detect
local t3 = os.time({year=2000, month=7, day=1, hour=12, min=0, sec=0})
-- t3 should equal either t1 or t2 depending on auto-detection
assert(t3 == t1 or t3 == t2, "auto-detected time should match one of the isdst variants")

print("OK")
