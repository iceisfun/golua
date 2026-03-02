-- Test: attrib.lua - Logical operator semantics
-- From: attrib.lua
-- What: Tests that and/or return proper values, not just true/false

do
  local a, b, c, d = 1 and nil, 1 or nil, (1 and (nil or 1)), 6
  assert(not a and b and c and d==6)
  assert((10 and 2) == 2)
  assert((10 or 2) == 10)
  assert((10 or assert(nil)) == 10)
  assert(not (nil and assert(nil)))
  assert((nil or "alo") == "alo")
  assert((nil and 10) == nil)
  assert((false and 10) == false)
  assert((true or 10) == true)
  assert((false or 10) == 10)
  assert(false ~= nil)
  assert(nil ~= false)
  assert(not nil == true)
  assert(not not nil == false)
  assert(not not 1 == true)
end
