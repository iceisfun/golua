-- Test: table.unpack edge cases and overflow
-- From: sort.lua
-- What: Tests table.unpack with extreme integer indices (maxinteger, mininteger), "too many results" errors, and boundary access.

do
local unpack = table.unpack
local maxI = math.maxinteger
local minI = math.mininteger

local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

do
  local maxi = (1 << 31) - 1          -- maximum value for an int (usually)
  local mini = -(1 << 31)             -- minimum value for an int (usually)
  checkerror("too many results", unpack, {}, 0, maxi)
  checkerror("too many results", unpack, {}, 1, maxi)
  checkerror("too many results", unpack, {}, 0, maxI)
  checkerror("too many results", unpack, {}, 1, maxI)
  checkerror("too many results", unpack, {}, mini, maxi)
  checkerror("too many results", unpack, {}, -maxi, maxi)
  checkerror("too many results", unpack, {}, minI, maxI)
  unpack({}, maxi, 0)
  unpack({}, maxi, 1)
  unpack({}, maxI, minI)
  pcall(unpack, {}, 1, maxi + 1)
  local a, b = unpack({[maxi] = 20}, maxi, maxi)
  assert(a == 20 and b == nil)
  a, b = unpack({[maxi] = 20}, maxi - 1, maxi)
  assert(a == nil and b == 20)
  local t = {[maxI - 1] = 12, [maxI] = 23}
  a, b = unpack(t, maxI - 1, maxI); assert(a == 12 and b == 23)
  a, b = unpack(t, maxI, maxI); assert(a == 23 and b == nil)
  a, b = unpack(t, maxI, maxI - 1); assert(a == nil and b == nil)
  t = {[minI] = 12.3, [minI + 1] = 23.5}
  a, b = unpack(t, minI, minI + 1); assert(a == 12.3 and b == 23.5)
  a, b = unpack(t, minI, minI); assert(a == 12.3 and b == nil)
  a, b = unpack(t, minI + 1, minI); assert(a == nil and b == nil)
end
end
