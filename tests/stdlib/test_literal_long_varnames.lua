-- Test: Long variable names
-- From: literals.lua
-- What: Tests that the compiler and VM correctly handle very long variable names (15000+ characters).

do
  local function dostring (x) return assert(load(x), "")() end

  local var1 = string.rep('a', 15000) .. '1'
  local var2 = string.rep('a', 15000) .. '2'
  local prog = string.format([[
    %s = 5
    %s = %s + 1
    return function () return %s - %s end
  ]], var1, var2, var1, var1, var2)
  local f = dostring(prog)
  assert(_G[var1] == 5 and _G[var2] == 6 and f() == -1)
  _G[var1], _G[var2] = nil
end
