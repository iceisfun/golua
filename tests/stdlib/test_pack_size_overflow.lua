-- Test: Format size overflow
-- From: tpack.lua
-- What: Tests that excessively large size digits in format strings produce "invalid format" errors, and tests packsize near the 2^31 limit.

do
local packsize = string.packsize

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

-- overflow in option size  (error will be in digit after limit)
checkerror("invalid format", packsize, "c1" .. string.rep("0", 40))

if packsize("i") == 4 then
  -- result would be 2^31  (2^3 repetitions of 2^28 strings)
  local s = string.rep("c268435456", 2^3)
  checkerror("too large", packsize, s)
  -- one less is OK
  s = string.rep("c268435456", 2^3 - 1) .. "c268435455"
  assert(packsize(s) == 0x7fffffff)
end
end
