-- Stress test: table mutation during iteration and metamethod execution

-- 1. Deleting keys during iteration
local t = {a=1, b=2, c=3, d=4, e=5}
local count = 0
for k, v in pairs(t) do
    count = count + 1
    t.a = nil
    t.b = nil
end

-- 2. Rehash during iteration (adding keys forces growth)
local big_t = {start = true}
local ok, err = pcall(function()
    for k, v in pairs(big_t) do
        for i = 1, 100 do
            big_t["new_" .. i] = i
        end
    end
end)

-- 3. next() correctly skips nil holes
local holey = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
for i = 1, 10, 2 do holey[i] = nil end

local hole_count = 0
for k, v in pairs(holey) do
    hole_count = hole_count + 1
end
assert(hole_count == 5, "next failed to skip deleted array holes")
