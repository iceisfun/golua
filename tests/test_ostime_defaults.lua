-- Test os.time default hour=12 and normalization of out-of-range month/day

-- Bug 1: hour should default to 12 when not specified
local t = os.time({year=2024, month=1, day=1})
local d = os.date("*t", t)
assert(d.hour == 12, "expected hour=12, got " .. tostring(d.hour))

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

print("PASS")
