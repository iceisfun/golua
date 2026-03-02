-- Test: nextvar.lua - Table hash fill and length operator
-- From: nextvar.lua
-- What: Tests filling table hash part with string keys, clearing them, then filling with numeric keys while checking the length operator.

do
  local a = {}

  -- make sure table has lots of space in hash part
  for i=1,100 do a[i.."+"] = true end
  for i=1,100 do a[i.."+"] = undef end
  -- fill hash part with numeric indices testing size operator
  for i=1,100 do
    a[i] = true
    assert(#a == i)
  end
end
