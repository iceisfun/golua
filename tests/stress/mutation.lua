print("--- Starting Table Mutation Probe ---")

local t = {a=1, b=2, c=3, d=4, e=5}

-- 1. Deleting the NEXT key in the sequence
-- Tests if your 'next' pointer gets lost when a bucket is cleared.
print("Testing deletion during iteration...")
local count = 0
for k, v in pairs(t) do
    count = count + 1
    t.a = nil
    t.b = nil
end
print("✓ Iteration survived mass-deletion")

-- 2. The "Rehash" Stressor
-- We add many keys to force a map growth/rehash while iterating.
print("\nTesting growth/rehash during iteration...")
local big_t = {start = true}
local ok, err = pcall(function()
    for k, v in pairs(big_t) do
        -- Add 100 new keys on the first iteration
        for i = 1, 100 do
            big_t["new_" .. i] = i
        end
    end
end)

if ok then
    print("✓ VM survived table rehash during pairs()")
else
    print("!! Potential Issue: " .. tostring(err))
end

-- 3. The "Nil Hole" iterator
-- Test if next() correctly skips holes created by deletions
local holey = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
for i = 1, 10, 2 do holey[i] = nil end -- delete odd indices

local hole_count = 0
for k, v in pairs(holey) do
    hole_count = hole_count + 1
end
assert(hole_count == 5, "Next failed to skip deleted array holes")
print("✓ Array-hole iteration verified")

print("\n--- Mutation Probe Complete ---")
