-- Test: calls.lua - Method declarations and self
-- From: calls.lua
-- What: Tests colon syntax for method declaration and the self parameter

do
  local a = {i = 10}
  local self = 20
  function a:x (x) return x+self.i end
  function a.y (x) return x+self end
  assert(a:x(1)+10 == a.y(1))

  a.t = {i=-100}
  a["t"].x = function (self, a,b) return self.i+a+b end
  assert(a.t:x(2,3) == -95)
end
