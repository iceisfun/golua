-- Test: pm.lua - gmatch init parameter
-- From: pm.lua
-- What: Tests the init (starting position) parameter for string.gmatch, including positive, negative, and boundary positions.

do   -- init parameter in gmatch
  local s = 0
  for k in string.gmatch("10 20 30", "%d+", 3) do
    s = s + tonumber(k)
  end
  assert(s == 50)

  s = 0
  for k in string.gmatch("11 21 31", "%d+", -4) do
    s = s + tonumber(k)
  end
  assert(s == 32)

  -- there is an empty string at the end of the subject
  s = 0
  for k in string.gmatch("11 21 31", "%w*", 9) do
    s = s + 1
  end
  assert(s == 1)

  -- there are no empty strings after the end of the subject
  s = 0
  for k in string.gmatch("11 21 31", "%w*", 10) do
    s = s + 1
  end
  assert(s == 0)
end
