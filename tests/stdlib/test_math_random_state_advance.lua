-- Test: every math.random call advances the generator by one step
-- What: reference math_random draws its value before it looks at the
-- arguments, so random(m, m) and calls that raise an argument error still
-- consume one value; seeded sequences must stay in step

local function after(f, ...)
  math.randomseed(7, 9)
  pcall(f, ...)
  return math.random(0)
end

assert(after(function() end) == 4637322584265719448)

local advanced = -5837484342485337481
assert(after(math.random) == advanced)
assert(after(math.random, 1, 1) == advanced)
assert(after(math.random, 5, 5) == advanced)
assert(after(math.random, 2, 1) == advanced)
assert(after(math.random, 1.5) == advanced)
assert(after(math.random, 1, 2, 3) == advanced)
assert(after(math.random, 0) == advanced)
