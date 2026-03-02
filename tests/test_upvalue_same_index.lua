-- Test: attrib.lua - Upvalue assignment with 5.2 beta bug
-- From: attrib.lua
-- What: Local and upvalue having the same index should work correctly

do
  local function foo ()
    local a
    return function ()
      local b
      a, b = 3, 14
      return a, b
    end
  end
  local a, b = foo()()
  assert(a == 3 and b == 14)
end
