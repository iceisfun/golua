-- test_math_ult: math.ult (unsigned less than)

assert(math.ult(1, 2) == true, "ult(1, 2)")
assert(math.ult(-1, -2) == false, "ult(-1, -2)")
assert(math.ult(-1, 2) == false, "ult(-1, 2) — -1 as unsigned is huge")
