-- Test: big.lua - Yields during large constant table access
-- From: big.lua
-- What: Tests that metamethod yields work during access to large tables (RK overflow)

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
  setmetatable(env, {
    __index = function (t, n) coroutine.yield('g'); return _G[n] end,
    __newindex = function (t, n, v) coroutine.yield('s'); _G[n] = v end,
  })
  X = nil
  local co = coroutine.wrap(f)
  assert(co() == 's')
  assert(co() == 'g')
  assert(co() == 'g')
  assert(co() == 0)
  assert(X[lim] == lim - 1 and X[lim + 1] == lim)
end
