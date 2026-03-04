-- Test: nextvar.lua - ipairs basic
-- From: nextvar.lua
-- What: Tests ipairs iteration over a table with array and hash parts, verifying it only iterates the array portion.

do
  local x = 0
  for k,v in ipairs{10,20,30;x=12} do
    x = x + 1
    assert(k == x and v == x * 10)
  end

  for _ in ipairs{x=12, y=24} do assert(nil) end
end
