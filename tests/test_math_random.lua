-- test_math_random: math.random and math.randomseed

-- Basic random [0, 1)
do
    local r = math.random()
    assert(r >= 0 and r < 1, string.format("random() out of range: %f", r))
end

-- Deterministic integer ranges
do
    local rands = {}
    math.randomseed(5)
    for i = 1, 20 do
        rands[i] = math.random(i, 2*i)
    end

    math.randomseed(5)
    local out, diff = 0, 0
    for i = 1, 20 do
        local r = math.random(i, 2*i)
        if r < i or r > 2*i then
            out = out + 1
        end
        if rands[i] ~= r then
            diff = diff + 1
        end
    end
    assert(out == 0 and diff == 0, string.format("seeded range: out=%d diff=%d", out, diff))
end

-- Degenerate range
assert(math.random(5, 5) == 5, "random(5,5) should be 5")

-- Negative range
do
    local t={}
    for i = 1, 100 do
        t[math.random(-2,-1)] = true
    end
    assert(t[-2] and t[-1], "random(-2,-1) should cover both values")
end

-- Empty range should error
assert(not pcall(math.random, 2, 1), "random(2,1) should error")

-- Large positive range
assert(math.random(0, math.maxinteger) >= 0, "random(0, maxinteger)")

-- math.random(0) should return a random integer (Lua 5.4)
do
    local a = math.random(0)
    local b = math.random(0)
    assert(a ~= b, "random(0) should return different values each call")
end

-- Large spanning range
assert(math.random(-1, math.maxinteger) >= -1, "random(-1, maxinteger)")

-- Full 64-bit range [mininteger, maxinteger] must not panic
do
    local min = math.mininteger
    local max = math.maxinteger

    local ok, val = pcall(math.random, min, max)
    assert(ok, "math.random(mininteger, maxinteger) panicked: " .. tostring(val))
    assert(type(val) == "number", "expected number from full-range random")

    -- Reproducibility with seeded full-range random
    math.randomseed(42, 0)
    local r1 = math.random(min, max)
    math.randomseed(42, 0)
    local r2 = math.random(min, max)
    assert(r1 == r2, "seeded full-range random should be reproducible")

    -- Single-arg form with maxinteger
    local ok3, val3 = pcall(math.random, max)
    assert(ok3, "math.random(maxinteger) panicked: " .. tostring(val3))
    assert(val3 >= 1 and val3 <= max, "math.random(maxinteger) out of range")

    -- Equal bounds edge cases
    assert(math.random(min, min) == min, "math.random(mininteger, mininteger) should return mininteger")
    assert(math.random(max, max) == max, "math.random(maxinteger, maxinteger) should return maxinteger")
    assert(math.random(1) == 1, "math.random(1) should return 1")
end

-- Seed/determinism
do
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
end

-- randomseed return values
do
    local function expect_two_numbers(...)
        local ok, a, b = ...
        assert(ok, select(2, ...))
        assert(type(a) == "number" and type(b) == "number", "expected math.randomseed to return two numbers")
    end

    expect_two_numbers(pcall(math.randomseed))
    expect_two_numbers(pcall(math.randomseed, 12345))
    expect_two_numbers(pcall(math.randomseed, 111, 222))
end

-- Seed reproducibility
do
    local function capture_sequence(seed1, seed2)
        math.randomseed(seed1, seed2)
        return {math.random(), math.random(), math.random()}
    end

    local _, s1, s2 = pcall(math.randomseed)
    assert(type(s1) == "number" and type(s2) == "number", "no-arg randomseed must return two numbers")

    local seq1 = capture_sequence(s1, s2)
    local seq2 = capture_sequence(s1, s2)
    for i = 1, #seq1 do
        assert(seq1[i] == seq2[i], string.format("expected deterministic random sequence, got %f vs %f", seq1[i], seq2[i]))
    end
end

-- pcall on nil should return false, not crash
assert(not pcall(rand), "pcall(rand) should return false for nil global")

-- Interval error still works
local ok2, err2 = pcall(math.random, 10, 1)
assert(not ok2, "math.random(10, 1) should error")
