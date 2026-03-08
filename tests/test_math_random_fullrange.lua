-- math.random(minint, maxint) should produce different values than math.random(0)
-- In Lua 5.4, random(0) uses direct I2UInt mapping while
-- random(minint, maxint) uses the project() function with rejection sampling.
-- GoLua incorrectly treats both as the same operation.

math.randomseed(42)
local r0 = math.random(0)

math.randomseed(42)
local rm = math.random(math.mininteger, math.maxinteger)

-- random(0) and random(minint, maxint) should produce DIFFERENT values
-- because they use different code paths in Lua 5.4
assert(r0 ~= rm, "random(0) and random(minint, maxint) should differ but both returned " .. r0)

-- Verify expected values from Lua 5.4 reference
assert(r0 == -1276290044721465627,
  "random(0) with seed 42 should be -1276290044721465627, got " .. r0)
assert(rm == 7947081992133310181,
  "random(minint,maxint) with seed 42 should be 7947081992133310181, got " .. rm)

-- After each call, the next random() should be the same
-- (both consume exactly 1 RNG draw)
math.randomseed(42)
math.random(0)
local after_0 = math.random()

math.randomseed(42)
math.random(math.mininteger, math.maxinteger)
local after_full = math.random()

assert(after_0 == after_full,
  "RNG state after random(0) and random(minint,maxint) should be identical")

print("OK")
