-- Test: math.lua - Minus zero and NaN
-- From: math.lua
-- What: Tests -0.0 equality with 0.0, distinguishing via 1/mz, NaN self-inequality, NaN comparison operators, and NaN as table key (rawset should fail).

do
  print("testing -0 and NaN")
  local mz <const> = -0.0
  local z <const> = 0.0
  assert(mz == z)
  assert(1/mz < 0 and 0 < 1/z)
  local a = {[mz] = 1}
  assert(a[z] == 1 and a[mz] == 1)
  a[z] = 2
  assert(a[z] == 2 and a[mz] == 2)
  local inf = math.huge * 2 + 1
  local mz <const> = -1/inf
  local z <const> = 1/inf
  assert(mz == z)
  assert(1/mz < 0 and 0 < 1/z)
  local NaN <const> = inf - inf
  assert(NaN ~= NaN)
  assert(not (NaN < NaN))
  assert(not (NaN <= NaN))
  assert(not (NaN > NaN))
  assert(not (NaN >= NaN))
  assert(not (0 < NaN) and not (NaN < 0))
  local NaN1 <const> = 0/0
  assert(NaN ~= NaN1 and not (NaN <= NaN1) and not (NaN1 <= NaN))
  local a = {}
  assert(not pcall(rawset, a, NaN, 1))
  assert(a[NaN] == undef)
  a[1] = 1
  assert(not pcall(rawset, a, NaN, 1))
  assert(a[NaN] == undef)
  -- strings with same binary representation as 0.0 (might create problems
  -- for constant manipulation in the pre-compiler)
  local a1, a2, a3, a4, a5 = 0, 0, "\0\0\0\0\0\0\0\0", 0, "\0\0\0\0\0\0\0\0"
  assert(a1 == a2 and a2 == a4 and a1 ~= a3)
  assert(a3 == a5)
end
