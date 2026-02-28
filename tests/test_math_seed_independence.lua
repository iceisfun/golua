-- test_math_jit_seeds.lua
-- math.randomseed should return seeds usable directly for reproducible sequences.

local function capture_sequence(seed1, seed2)
    math.randomseed(seed1, seed2)
    return {math.random(), math.random(), math.random()}
end

-- Seeding with zero arguments should give a valid pair
local _, s1, s2 = pcall(math.randomseed)
assert(type(s1) == "number" and type(s2) == "number", "no-arg randomseed must return two numbers")

local seq1 = capture_sequence(s1, s2)
local seq2 = capture_sequence(s1, s2)
for i = 1, #seq1 do
    assert(seq1[i] == seq2[i], string.format("expected deterministic random sequence, got %f vs %f", seq1[i], seq2[i]))
end
