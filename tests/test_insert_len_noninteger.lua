-- Test: table.insert __len non-integer error
-- From: sort.lua
-- What: Tests that table.insert raises an error when the object's __len metamethod returns a non-integer value.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

do   -- length is not an integer
  local t = setmetatable({}, {__len = function () return 'abc' end})
  assert(#t == 'abc')
  checkerror("object length is not an integer", table.insert, t, 1)
end
end
