-- Test: math.lua - Trigonometric and math library functions
-- From: math.lua
-- What: Tests sin, cos, tan, atan, acos, asin, deg, rad, abs, fmod, sqrt, log, and exp for correctness.

do
  local minint <const> = math.mininteger
  local maxint <const> = math.maxinteger

  local floatbits = 24
  do
    local p = 2.0^floatbits
    while p < p + 1.0 do
      p = p * 2.0
      floatbits = floatbits + 1
    end
  end

  local function eq (a,b,limit)
    if not limit then
      if floatbits >= 50 then limit = 1E-11
      else limit = 1E-5
      end
    end
    return a == b or math.abs(a-b) <= limit
  end

  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  assert(eq(math.sin(-9.8)^2 + math.cos(-9.8)^2, 1))
  assert(eq(math.tan(math.pi/4), 1))
  assert(eq(math.sin(math.pi/2), 1) and eq(math.cos(math.pi/2), 0))
  assert(eq(math.atan(1), math.pi/4) and eq(math.acos(0), math.pi/2) and
         eq(math.asin(1), math.pi/2))
  assert(eq(math.deg(math.pi/2), 90) and eq(math.rad(90), math.pi/2))
  assert(math.abs(-10.43) == 10.43)
  assert(eqT(math.abs(minint), minint))
  assert(eqT(math.abs(maxint), maxint))
  assert(eqT(math.abs(-maxint), maxint))
  assert(eq(math.atan(1,0), math.pi/2))
  assert(math.fmod(10,3) == 1)
  assert(eq(math.sqrt(10)^2, 10))
  assert(eq(math.log(2, 10), math.log(2)/math.log(10)))
  assert(eq(math.log(2, 2), 1))
  assert(eq(math.log(9, 3), 2))
  assert(eq(math.exp(0), 1))
  assert(eq(math.sin(10), math.sin(10%(2*math.pi))))
end
