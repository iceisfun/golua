-- ==========================================================================
-- Fengari test extraction: Math library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: math
-- Total tests: 12
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] math.abs, math.sin, math.cos, math.tan, math.asin, math.acos, math.atan
-- Verifies: output matches expected value via print()
do
  print(math.abs(-10), math.abs(-10.5), math.cos(10), math.tan(10), math.asin(1), math.acos(0.5), math.atan(10))
end
--> =10	10.5	-0.8390715290764524	0.6483608274590866	1.5707963267948966	1.0471975511965979	1.4711276743037347

-- --------------------------------------------------------------------------
-- [Test 2] math.ceil, math.floor
-- Verifies: output matches expected value via print()
do
  print(math.ceil(10.5), math.floor(10.5))
end
--> =11	10

-- --------------------------------------------------------------------------
-- [Test 3] math.deg, math.rad
-- Verifies: output matches expected value via print()
do
  print(math.deg(10), math.rad(10))
end
--> =572.9577951308232	0.17453292519943295

-- --------------------------------------------------------------------------
-- [Test 4] math.log
-- Verifies: output matches expected value via print()
do
  print(math.log(10), math.log(10, 2), math.log(10, 10))
end
--> =2.302585092994046	3.321928094887362	1

-- --------------------------------------------------------------------------
-- [Test 5] math.min, math.max
-- Verifies: output matches expected value via print()
do
  print(math.max(10, 5, 23), math.min(10, 5, 23))
end
--> =23	5

-- --------------------------------------------------------------------------
-- [Test 6] math.random
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print(math.random(), math.random(10, 15))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 7] math.sqrt
-- Verifies: output matches expected value via print()
do
  print(math.sqrt(10))
end
--> =3.1622776601683795

-- --------------------------------------------------------------------------
-- [Test 8] math.tointeger
-- Verifies: output matches expected value via print()
do
  print(math.tointeger('10'))
end
--> =10

-- --------------------------------------------------------------------------
-- [Test 9] math.type
-- Verifies: output matches expected value via print()
do
  print(math.type(10), math.type(10.5), math.type('hello'))
end
--> =integer	float

-- --------------------------------------------------------------------------
-- [Test 10] math.ult
-- Verifies: output matches expected value via print()
do
  print(math.ult(5, 200))
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 11] math.fmod
-- Verifies: output matches expected value via print()
do
  print(math.fmod(2,5))
end
--> =2

-- --------------------------------------------------------------------------
-- [Test 12] math.modf
-- Verifies: output matches expected value via print()
do
  print(math.modf(3.4, 0.6))
end
--> =3	0.3999999999999999
