-- Test: events.lua - Environment metatable (__index, __newindex)
-- From: events.lua
-- What: Tests using metatables on _ENV for __index and __newindex, verifying environment isolation

do
  X = 20; B = 30
  _ENV = setmetatable({}, {__index=_G})
  X = X+10
  assert(X == 30 and _G.X == 20)
  B = false
  assert(B == false)
  _ENV["B"] = undef
  assert(B == 30)
  _G.X, _G.B = nil
end
