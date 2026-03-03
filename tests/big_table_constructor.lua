-- Test: big.lua - Large table constructor
-- From: big.lua
-- What: Creates a table with 2^18+1000 elements via generated code, tests that values are correct

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
  f()
  assert(env.X[lim] == lim - 1 and env.X[lim + 1] == lim)
end
