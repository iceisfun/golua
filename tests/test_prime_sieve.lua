-- Prime sieve: stresses table allocation, iterative table-set, and math.sqrt.

local function assert_eq(a, b, msg)
    if a ~= b then error(msg or (tostring(a) .. " ~= " .. tostring(b)), 2) end
end

local function sieve(limit)
    local primes = {}
    for i = 2, limit do
        primes[i] = true
    end

    for i = 2, math.sqrt(limit) do
        if primes[i] then
            for j = i * i, limit, i do
                primes[j] = false
            end
        end
    end

    local results = {}
    for i = 2, limit do
        if primes[i] then
            table.insert(results, i)
        end
    end
    return results
end

-- 1. Known prime counts
assert_eq(#sieve(10), 4, "primes up to 10")
assert_eq(#sieve(100), 25, "primes up to 100")
assert_eq(#sieve(1000), 168, "primes up to 1000")
assert_eq(#sieve(10000), 1229, "primes up to 10000")

-- 2. First few primes are correct
local p = sieve(30)
local expected = {2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
assert_eq(#p, #expected, "prime count up to 30")
for i = 1, #expected do
    assert_eq(p[i], expected[i], "prime #" .. i .. " up to 30")
end

-- 3. Sieve of 50k (5133 primes) exercises large table capacity
assert_eq(#sieve(50000), 5133, "primes up to 50000")

-- 4. Edge case: limit below any prime
assert_eq(#sieve(1), 0, "primes up to 1")
assert_eq(#sieve(2), 1, "primes up to 2")
assert_eq(#sieve(3), 2, "primes up to 3")
