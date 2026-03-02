-- Test: table.move overflow and wrap-around errors
-- From: sort.lua
-- What: Tests that table.move produces correct errors for "too many" elements and "wrap around" conditions with extreme indices.

do
local maxI = math.maxinteger
local minI = math.mininteger

local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

checkerror("too many", table.move, {}, 0, maxI, 1)
checkerror("too many", table.move, {}, -1, maxI - 1, 1)
checkerror("too many", table.move, {}, minI, -1, 1)
checkerror("too many", table.move, {}, minI, maxI, 1)
checkerror("wrap around", table.move, {}, 1, maxI, 2)
checkerror("wrap around", table.move, {}, 1, 2, maxI)
checkerror("wrap around", table.move, {}, minI, -2, 2)
end
