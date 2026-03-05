-- Test: os.date("!*t") returns UTC time, not local time
-- Bug: the "!" prefix was stripped to detect "*t" but UTC flag was not
-- passed to DateTable, so it always returned local time.

do
  local t = 1000000000 -- 2001-09-09 01:46:40 UTC

  -- String format with ! gives UTC
  local utc_str = os.date("!%Y-%m-%d %H:%M:%S", t)
  assert(utc_str == "2001-09-09 01:46:40",
    "os.date('!format') should return UTC: got " .. utc_str)

  -- Table format with ! should also give UTC
  local utc_tbl = os.date("!*t", t)
  assert(utc_tbl.year == 2001, "UTC year should be 2001, got " .. utc_tbl.year)
  assert(utc_tbl.month == 9, "UTC month should be 9, got " .. utc_tbl.month)
  assert(utc_tbl.day == 9, "UTC day should be 9, got " .. utc_tbl.day)
  assert(utc_tbl.hour == 1, "UTC hour should be 1, got " .. utc_tbl.hour)
  assert(utc_tbl.min == 46, "UTC min should be 46, got " .. utc_tbl.min)
  assert(utc_tbl.sec == 40, "UTC sec should be 40, got " .. utc_tbl.sec)

  -- Non-UTC table should give local time (different from UTC unless in UTC zone)
  local local_tbl = os.date("*t", t)
  -- Just verify the table is valid
  assert(type(local_tbl.year) == "number")
  assert(type(local_tbl.hour) == "number")

  -- The UTC and local tables should have the same second but potentially different hours
  -- (unless running in UTC timezone)
  assert(utc_tbl.sec == local_tbl.sec, "seconds should match")

  print("PASS: os.date('!*t') returns UTC time")
end
