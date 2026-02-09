-- test_math_random2: math.random and math.randomseed edge cases

-- Two calls to randomseed() with no args should produce different sequences
math.randomseed()
local r1 = math.random()
math.randomseed()
local r2 = math.random()
assert(r1 ~= r2, "two randomseed() calls should produce different sequences")

-- Saving and restoring seeds should reproduce the sequence
local s1, s2 = math.randomseed()
r1 = math.random()
math.randomseed(s1, s2)
r2 = math.random()
assert(r1 == r2, "restored seed should reproduce sequence")

-- Bad seed args should error
assert(not pcall(math.randomseed, "hi"), "randomseed('hi') should error")
assert(not pcall(math.randomseed, 1, {}), "randomseed(1, {}) should error")
assert(not pcall(math.randomseed, 1.1, 2), "randomseed(1.1, 2) should error")

-- pcall on nil should return false, not crash
assert(not pcall(rand), "pcall(rand) should return false for nil global")

-- random() returns [0, 1)
local r = math.random()
assert(r >= 0 and r < 1, string.format("random() out of range: %f", r))

-- Deterministic integer ranges
local rands = {}
math.randomseed(5)
for i = 1, 20 do
    rands[i] = math.random(i, 2*i)
end

math.randomseed(5)
local out, diff = 0, 0
for i = 1, 20 do
    r = math.random(i, 2*i)
    if r < i or r > 2*i then
        out = out + 1
    end
    if rands[i] ~= r then
        diff = diff + 1
    end
end
assert(out == 0 and diff == 0, string.format("seeded range: out=%d diff=%d", out, diff))

-- Degenerate range
assert(math.random(5, 5) == 5, "random(5,5) should be 5")

-- Negative range
local t={}
for i = 1, 100 do
    t[math.random(-2,-1)] = true
end
assert(t[-2] and t[-1], "random(-2,-1) should cover both values")

-- Empty range should error
assert(not pcall(math.random, 2, 1), "random(2,1) should error")

-- Large positive range
assert(math.random(0, math.maxinteger) >= 0, "random(0, maxinteger)")

-- math.random(0) should return a random integer (Lua 5.4)
local a = math.random(0)
local b = math.random(0)
assert(a ~= b, "random(0) should return different values each call")

-- Large spanning range
assert(math.random(-1, math.maxinteger) >= -1, "random(-1, maxinteger)")
