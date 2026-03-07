-- ==========================================================================
-- Fengari test extraction: Math library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: math
-- Total tests: 12
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.

-- [Test 1] math.abs, math.sin, math.cos, math.tan, math.asin, math.acos, math.atan
do
  assert(math.abs(-10) == 10)
  assert(math.abs(-10.5) == 10.5)
  assert(math.cos(10) - (-0.8390715290764524) < 1e-12)
  assert(math.tan(10) - 0.6483608274590866 < 1e-12)
  assert(math.asin(1) - 1.5707963267948966 < 1e-12)
  assert(math.acos(0.5) - 1.0471975511965979 < 1e-12)
  assert(math.atan(10) - 1.4711276743037347 < 1e-12)
end

-- --------------------------------------------------------------------------
-- [Test 2] math.ceil, math.floor
do
  assert(math.ceil(10.5) == 11)
  assert(math.floor(10.5) == 10)
end

-- --------------------------------------------------------------------------
-- [Test 3] math.deg, math.rad
do
  assert(math.deg(10) - 572.9577951308232 < 1e-6)
  assert(math.rad(10) - 0.17453292519943295 < 1e-12)
end

-- --------------------------------------------------------------------------
-- [Test 4] math.log
do
  assert(math.log(10) - 2.302585092994046 < 1e-12)
  assert(math.log(10, 2) - 3.321928094887362 < 1e-12)
  assert(math.log(10, 10) == 1.0)
end

-- --------------------------------------------------------------------------
-- [Test 5] math.min, math.max
do
  assert(math.max(10, 5, 23) == 23)
  assert(math.min(10, 5, 23) == 5)
end

-- --------------------------------------------------------------------------
-- [Test 6] math.random
do
  local r1 = math.random()
  assert(r1 >= 0 and r1 < 1)
  local r2 = math.random(10, 15)
  assert(r2 >= 10 and r2 <= 15 and math.type(r2) == "integer")
end

-- --------------------------------------------------------------------------
-- [Test 7] math.sqrt
do
  assert(math.sqrt(10) - 3.1622776601683795 < 1e-12)
end

-- --------------------------------------------------------------------------
-- [Test 8] math.tointeger
do
  assert(math.tointeger(10) == 10)
  assert(math.tointeger(10.0) == 10)
  assert(math.tointeger('10') == 10)
end

-- --------------------------------------------------------------------------
-- [Test 9] math.type
do
  assert(math.type(10) == "integer")
  assert(math.type(10.5) == "float")
  assert(not math.type('hello'))
end

-- --------------------------------------------------------------------------
-- [Test 10] math.ult
do
  assert(math.ult(5, 200) == true)
end

-- --------------------------------------------------------------------------
-- [Test 11] math.fmod
do
  assert(math.fmod(2,5) == 2)
end

-- --------------------------------------------------------------------------
-- [Test 12] math.modf
do
  local i, f = math.modf(3.4, 0.6)
  assert(i == 3)
  assert(f - 0.4 < 1e-12)
end
