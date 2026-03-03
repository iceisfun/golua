-- Test: files.lua - os.difftime
-- From: files.lua
-- What: Tests time difference calculations

do
  local t1 = os.time{year=2000, month=10, day=1, hour=23, min=12}
  local t2 = os.time{year=2000, month=10, day=1, hour=23, min=10, sec=19}
  assert(os.difftime(t1,t2) == 60*2-19)
end
