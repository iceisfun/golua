-- Test: files.lua - os.time normalization
-- From: files.lua
-- What: Tests that os.time normalizes out-of-range fields (since 5.3.3)

do
  local t1 = {year = 2005, month = 1, day = 1, hour = 1, min = 0, sec = -3602}
  os.time(t1)
  assert(t1.day == 31 and t1.month == 12 and t1.year == 2004 and
         t1.hour == 23 and t1.min == 59 and t1.sec == 58 and
         t1.yday == 366)
end
