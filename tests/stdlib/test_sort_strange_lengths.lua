-- Test: table.sort with strange lengths
-- From: sort.lua
-- What: Tests table.sort behavior when __len returns -1 (empty sort) and when __len returns maxinteger (should error "too big").

do
local maxI = math.maxinteger

local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

local a = setmetatable({}, {__len = function () return -1 end})
assert(#a == -1)
table.sort(a, error)    -- should not compare anything
a = setmetatable({}, {__len = function () return maxI end})
checkerror("too big", table.sort, a)
end
