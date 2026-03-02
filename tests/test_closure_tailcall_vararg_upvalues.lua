-- Test: closure.lua - Closing upvalues in tail calls of vararg functions
-- From: closure.lua
-- What: Tests correct closing of upvalues during tail calls from vararg functions

do
  local function t ()
    local function c(a,b) assert(a=="test" and b=="OK") end
    local function v(f, ...) c("test", f() ~= 1 and "FAILED" or "OK") end
    local x = 1
    return v(function() return x end)
  end
  t()
end
