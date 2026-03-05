-- Test: big.lua - Errors in metamethods for large tables
-- From: big.lua
-- What: Tests error handling when metamethods fail during large table access

do
  local lim = 2^18 + 1000
  local prog = { "local y = {0" }
  for i = 1, lim do prog[#prog + 1] = i  end
  prog[#prog + 1] = "}\n"
  prog[#prog + 1] = "X = y\n"
  prog[#prog + 1] = ("assert(X[%d] == %d)"):format(lim - 1, lim - 2)
  prog[#prog + 1] = "return 0"
  prog = table.concat(prog, ";")
  local env = {string = string, assert = assert}
  local f = assert(load(prog, nil, nil, env))

  local undef = nil
  for k in pairs(env) do env[k] = undef end
  setmetatable(env, {})

  getmetatable(env).__index = function () end
  getmetatable(env).__newindex = function () end
  local e, m = pcall(f)
  assert(not e and m:find("global 'X'"))

  getmetatable(env).__newindex = function () error("hi") end
  local e, m = xpcall(f, debug.traceback)
  assert(not e and m:find("'newindex'"))
end
