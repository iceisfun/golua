-- Test os.time default hour=12 and normalization of out-of-range month/day

-- Bug 1: hour should default to 12 when not specified
local t = os.time({year=2024, month=1, day=1})
local d = os.date("*t", t)
assert(d.hour == 12, "expected hour=12, got " .. tostring(d.hour))

-- Defaults should also include min=0 and sec=0, and write back to input table
local td = {year=2024, month=2, day=3}
local _ = os.time(td)
assert(td.hour == 12, "default table writeback: expected hour=12, got " .. tostring(td.hour))
assert(td.min == 0, "default table writeback: expected min=0, got " .. tostring(td.min))
assert(td.sec == 0, "default table writeback: expected sec=0, got " .. tostring(td.sec))

-- Bug 2: month=0 should normalize to December of previous year
local t1 = os.time({year=2024, month=0, day=1, hour=12, min=0, sec=0})
local d1 = os.date("!*t", t1)
assert(d1.year == 2023, "month=0: expected year=2023, got " .. tostring(d1.year))
assert(d1.month == 12, "month=0: expected month=12, got " .. tostring(d1.month))
assert(d1.day == 1, "month=0: expected day=1, got " .. tostring(d1.day))

-- Bug 2: day=0 should normalize to last day of previous month
local t2 = os.time({year=2024, month=1, day=0, hour=12, min=0, sec=0})
local d2 = os.date("!*t", t2)
assert(d2.year == 2023, "day=0: expected year=2023, got " .. tostring(d2.year))
assert(d2.month == 12, "day=0: expected month=12, got " .. tostring(d2.month))
assert(d2.day == 31, "day=0: expected day=31, got " .. tostring(d2.day))

-- Additional: month=-1 should go back two months
local t3 = os.time({year=2024, month=-1, day=1, hour=12, min=0, sec=0})
local d3 = os.date("!*t", t3)
assert(d3.year == 2023, "month=-1: expected year=2023, got " .. tostring(d3.year))
assert(d3.month == 11, "month=-1: expected month=11, got " .. tostring(d3.month))

-- Additional: day=-1 should go back two days from start of month
local t4 = os.time({year=2024, month=1, day=-1, hour=12, min=0, sec=0})
local d4 = os.date("!*t", t4)
assert(d4.year == 2023, "day=-1: expected year=2023, got " .. tostring(d4.year))
assert(d4.month == 12, "day=-1: expected month=12, got " .. tostring(d4.month))
assert(d4.day == 30, "day=-1: expected day=30, got " .. tostring(d4.day))

-- Additional: month overflow should normalize forward
local t5 = os.time({year=2024, month=13, day=1, hour=12, min=0, sec=0})
local d5 = os.date("!*t", t5)
assert(d5.year == 2025, "month=13: expected year=2025, got " .. tostring(d5.year))
assert(d5.month == 1, "month=13: expected month=1, got " .. tostring(d5.month))
assert(d5.day == 1, "month=13: expected day=1, got " .. tostring(d5.day))

-- Additional: day overflow should normalize forward
local t6 = os.time({year=2024, month=1, day=32, hour=12, min=0, sec=0})
local d6 = os.date("!*t", t6)
assert(d6.year == 2024, "day=32: expected year=2024, got " .. tostring(d6.year))
assert(d6.month == 2, "day=32: expected month=2, got " .. tostring(d6.month))
assert(d6.day == 1, "day=32: expected day=1, got " .. tostring(d6.day))

-- Additional: time components should normalize too
local t7 = os.time({year=2024, month=1, day=1, hour=24, min=60, sec=60})
local t7_expected = os.time({year=2024, month=1, day=2, hour=1, min=1, sec=0})
assert(t7 == t7_expected, "time overflow should normalize to next-day 01:01:00")

-- year=0 is a valid year (1 BC), should not error
local ok, val = pcall(function() return os.time{year=0,month=1,day=1} end)
assert(ok, "year=0 should be accepted, got: " .. tostring(val))

-- year=0 should round-trip and match explicit/default-time variants
local y0_default = os.time({year=0, month=1, day=1})
local y0_explicit = os.time({year=0, month=1, day=1, hour=12, min=0, sec=0})
assert(y0_default == y0_explicit, "year=0 default/explicit mismatch")
local y0 = os.date("!*t", y0_default)
assert(y0.year == 0, "year=0 round-trip: expected year=0, got " .. tostring(y0.year))
assert(y0.month == 1, "year=0 round-trip: expected month=1, got " .. tostring(y0.month))
assert(y0.day == 1, "year=0 round-trip: expected day=1, got " .. tostring(y0.day))

-- Missing required fields should still be rejected
local ok2, err2 = pcall(function() return os.time{month=1, day=1} end)
assert(not ok2 and tostring(err2):find("year"), "missing year should fail")

-- os.date with number arg should use Lua number-to-string coercion
assert(os.date(1.0, 0) == "1.0", "os.date(1.0) should produce '1.0'")
assert(os.date(-0.0, 0) == "-0.0", "os.date(-0.0) should produce '-0.0'")
assert(os.date(1/0, 0) == "inf", "os.date(inf) should produce 'inf'")
assert(os.date(0/0, 0) == "-nan", "os.date(nan) should produce '-nan'")

print("PASS")
