-- test_iterator: generic for-in with iterator functions

-- Basic iterator
do
    local function iter(t)
        local i = 0
        return function()
            i = i + 1
            return t[i]
        end
    end

    local t = { 1, 2, 3 }
    local out = {}

    for v in iter(t) do
        out[#out + 1] = v
    end

    assert(#out == 3)
    assert(out[1] == 1 and out[3] == 3)
end

-- Iterator state resets between separate loops
-- Regression test: the second for loop must start fresh (k=nil)
-- rather than resuming from the exhausted state of the first loop.
do
    local results = {}
    local tab = {a = 1, b = 2}

    local fn = function(s, k)
        local next_k, _ = next(tab, k)
        return next_k
    end

    for item in fn do
        results[#results + 1] = item
    end

    assert(#results == 2, "first loop: expected 2 items, got " .. #results)

    for item in fn do
        results[#results + 1] = item
    end

    assert(#results == 4, "both loops: expected 4 items, got " .. #results)

    for i = 1, #results do
        assert(tab[results[i]] ~= nil,
            "unexpected key: " .. tostring(results[i]))
    end
end
