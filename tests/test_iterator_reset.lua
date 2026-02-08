-- Test that generic for-in iterator state resets between separate loops.
-- Regression test from gopher-lua issue #514: the second for loop must
-- start fresh (k=nil) rather than resuming from the exhausted state of
-- the first loop.

local results = {}
local tab = {a = 1, b = 2}

local fn = function(s, k)
    local next_k, _ = next(tab, k)
    return next_k
end

for item in fn do
    results[#results + 1] = item
end

-- First loop should have yielded both keys
assert(#results == 2, "first loop: expected 2 items, got " .. #results)

for item in fn do
    results[#results + 1] = item
end

-- Second loop should also yield both keys (total 4)
assert(#results == 4, "both loops: expected 4 items, got " .. #results)

-- Verify all collected items are valid keys from the table
for i = 1, #results do
    assert(tab[results[i]] ~= nil,
        "unexpected key: " .. tostring(results[i]))
end
