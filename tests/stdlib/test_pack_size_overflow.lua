-- Test: Format size overflow
-- From: tpack.lua
-- What: Tests that excessively large size digits in format strings produce "invalid format" errors, and tests packsize near the LUA_MAXINTEGER limit.

do
local packsize = string.packsize

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

-- overflow in option size  (error will be in digit after limit)
checkerror("invalid format", packsize, "c1" .. string.rep("0", 40))

-- maxsize-9 is accepted; overflowing the total is rejected as "too large".
-- Mirrors official tpack.lua (5.5): the size limit is LUA_MAXINTEGER.
local maxsize = (packsize("j") <= packsize("T")) and
                math.maxinteger or (1 << (packsize("j") * 8 - 1)) - 1
assert(packsize(string.format("c%d", maxsize - 9)) == maxsize - 9)
checkerror("too large", packsize, string.format("c%dc10", maxsize - 9))
end
