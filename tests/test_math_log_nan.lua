-- math.log of negative numbers should produce -nan (matching C's log)
assert(tostring(math.log(-1)) == "-nan", "math.log(-1) should be -nan, got: " .. tostring(math.log(-1)))
assert(tostring(math.log(-2)) == "-nan", "math.log(-2) should be -nan, got: " .. tostring(math.log(-2)))
-- math.log with base that produces NaN
assert(tostring(math.log(10, -1)) == "-nan", "math.log(10,-1) should be -nan, got: " .. tostring(math.log(10, -1)))
