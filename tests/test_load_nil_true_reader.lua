-- Test: calls.lua - Load with nil/true reader returns
-- From: calls.lua
-- What: Tests load() edge cases with reader returning nil or true

do
  local a = assert(load(function () return nil end))
  a()  -- empty chunk
  assert(not load(function () return true end))
end
