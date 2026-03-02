-- Test: events.lua - Multi-level __call
-- From: events.lua
-- What: Tests chained __call metamethods through multiple levels

do
  local i
  local tt = {
    __call = function (t, ...)
      i = i+1
      if t.f then return t.f(...)
      else return {...}
      end
    end
  }

  local a = setmetatable({}, tt)
  local b = setmetatable({f=a}, tt)
  local c = setmetatable({f=b}, tt)

  i = 0
  local x = c(3,4,5)
  assert(i == 3 and x[1] == 3 and x[3] == 5)
end
