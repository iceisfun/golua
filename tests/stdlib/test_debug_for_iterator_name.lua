-- Test: db.lua - For-iterator name in debug info
-- From: db.lua
-- What: Tests that debug.getinfo reports for-iterator functions with name "for iterator"

do
  local function f()
    assert(debug.getinfo(1).name == "for iterator")
  end
  for i in f do end
end
