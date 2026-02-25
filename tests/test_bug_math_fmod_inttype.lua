-- Bug #5: math.fmod returns float for integer arguments
-- Lua 5.4 returns integer when both arguments are integers.

assert(math.fmod(7, 3) == 1, "fmod value should be correct")
assert(math.type(math.fmod(7, 3)) == "integer", "fmod(int, int) should return integer")
assert(math.type(math.fmod(10, 5)) == "integer", "fmod(int, int) should return integer")

-- Float args should remain float
assert(math.type(math.fmod(7.0, 3.0)) == "float", "fmod(float, float) should return float")
assert(math.type(math.fmod(7.0, 3)) == "float", "fmod(float, int) should return float")
