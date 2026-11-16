-- Test: load checks the mode argument before the chunk name (luaB_load calls
-- getMode before luaL_optstring on argument 2), so with both bad the error
-- names argument #3.
local t = {}
local function msg(...)
  local ok, err = pcall(load, ...)
  assert(not ok)
  return err
end
assert(msg(print, t, t):find("bad argument #3 to 'load' (string expected, got table)", 1, true))
assert(msg('x', t, t):find("bad argument #3 to 'load' (string expected, got table)", 1, true))
assert(msg('x', t, 'B'):find("bad argument #3 to 'load' (invalid mode)", 1, true))
assert(msg('x', t, 't'):find("bad argument #2 to 'load' (string expected, got table)", 1, true))
print("PASS")
