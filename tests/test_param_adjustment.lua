-- Test: calls.lua - Parameter adjustment
-- From: calls.lua
-- What: Tests correct nil-filling when functions receive fewer arguments than parameters

do
  assert((function () return nil end)(4) == nil)
  assert((function () local a; return a end)(4) == nil)
  assert((function (a) return a end)() == nil)
end
