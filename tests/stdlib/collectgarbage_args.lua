-- Test: collectgarbage validates its arguments like luaL_checkoption /
-- luaL_optinteger (GC behaviour itself is not under test).
local function msg(...)
  local ok, m = pcall(collectgarbage, ...)
  assert(not ok)
  return m
end

assert(msg({}) == "bad argument #1 to 'collectgarbage' (string expected, got table)")
assert(msg(1) == "bad argument #1 to 'collectgarbage' (invalid option '1')")
assert(msg('bogus') == "bad argument #1 to 'collectgarbage' (invalid option 'bogus')")

-- numeric strings are accepted for the parameter value
local old = collectgarbage('param', 'pause')
assert(math.type(collectgarbage('param', 'pause', '100')) == 'integer')
assert(collectgarbage('param', 'pause') == 100)
assert(msg('param', 'pause', 'x') == "bad argument #3 to 'collectgarbage' (number expected, got string)")
assert(msg('param', 'pause', 1.5) == "bad argument #3 to 'collectgarbage' (number has no integer representation)")
collectgarbage('param', 'pause', old)

-- the step size is validated too
assert(msg('step', 'x') == "bad argument #2 to 'collectgarbage' (number expected, got string)")
assert(msg('step', 1.5) == "bad argument #2 to 'collectgarbage' (number has no integer representation)")
assert(type(collectgarbage('step', '0x10')) == 'boolean')
assert(type(collectgarbage('step', 3)) == 'boolean')
print("PASS")
