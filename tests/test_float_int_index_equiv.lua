-- Test: attrib.lua - Large float/integer index equivalence
-- From: attrib.lua
-- What: Floats and integers that represent the same value must index the same table slots

do
  local a = {}
  local maxint = math.maxinteger
  while maxint ~= (maxint + 0.0) or (maxint - 1) ~= (maxint - 1.0) do
    maxint = maxint // 2
  end
  local maxintF = maxint + 0.0
  assert(maxintF == maxint and math.type(maxintF) == "float" and
         maxintF >= 2.0^14)

  a[maxintF] = 10; a[maxintF - 1.0] = 11;
  a[-maxintF] = 12; a[-maxintF + 1.0] = 13;
  assert(a[maxint] == 10 and a[maxint - 1] == 11 and
         a[-maxint] == 12 and a[-maxint + 1] == 13)
end
