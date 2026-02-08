-- Table stress tests: rehash during iteration, hole punching, and
-- iteration correctness after deletion.

--------------------------------------------------------------------------------
-- 1. Table growth during pairs()
-- Forces the table to rehash while an active iterator is scanning it.
-- Verifies that iteration does not abort prematurely and table state
-- remains consistent after mutation.
--------------------------------------------------------------------------------
local t = {}
for i = 1, 100 do t[i] = i end

local visited_count = 0

for k, v in pairs(t) do
    visited_count = visited_count + 1
    -- On the 50th item, flood the table with new keys to trigger rehash.
    if visited_count == 50 then
        for j = 200, 5200 do
            t[j] = "new_val"
        end
    end
end

assert(visited_count >= 50, "Test 1 failed: iterator aborted prematurely")
assert(t[5200] == "new_val", "Test 1 failed: table integrity lost after rehash")

--------------------------------------------------------------------------------
-- 2. "Hole Puncher" — rapid set/delete loop
-- Creates and immediately deletes even-numbered keys. This exercises the
-- table's array shrinking and hash deletion paths. The test verifies that
-- no "attempt to index a nil value" error occurs and that surviving keys
-- retain their correct values.
--------------------------------------------------------------------------------
local big_t = {}

for i = 1, 10000 do
    big_t[i] = i
    if i % 2 == 0 then
        big_t[i] = nil -- Punch a hole
    end
end

-- Verify odd keys survived, even keys are gone
for i = 1, 10000 do
    if i % 2 == 0 then
        assert(big_t[i] == nil, "Test 2 failed: deleted key " .. i .. " still present")
    else
        assert(big_t[i] == i, "Test 2 failed: key " .. i .. " has wrong value")
    end
end

--------------------------------------------------------------------------------
-- 3. Iteration after hole punching
-- Verifies that pairs() correctly skips nil-valued entries (holes) in the
-- array part after deletion. A naive Next() implementation that returns
-- nil-valued array slots would break iteration.
--------------------------------------------------------------------------------
local ht = {}
for i = 1, 10 do ht[i] = i * 10 end

-- Delete keys 2, 4, 6, 8
ht[2] = nil
ht[4] = nil
ht[6] = nil
ht[8] = nil

local seen = {}
for k, v in pairs(ht) do
    assert(v ~= nil, "Test 3 failed: pairs() returned nil value for key " .. tostring(k))
    seen[k] = v
end

-- Should see 1, 3, 5, 7, 9, 10
assert(seen[1] == 10, "Test 3 failed: key 1 missing or wrong")
assert(seen[3] == 30, "Test 3 failed: key 3 missing or wrong")
assert(seen[5] == 50, "Test 3 failed: key 5 missing or wrong")
assert(seen[7] == 70, "Test 3 failed: key 7 missing or wrong")
assert(seen[9] == 90, "Test 3 failed: key 9 missing or wrong")
assert(seen[10] == 100, "Test 3 failed: key 10 missing or wrong")
assert(seen[2] == nil, "Test 3 failed: deleted key 2 appeared in pairs()")
assert(seen[4] == nil, "Test 3 failed: deleted key 4 appeared in pairs()")

--------------------------------------------------------------------------------
-- 4. Hash-only churn
-- Uses large integer keys that bypass the array part entirely, exercising
-- the hash part's delete-on-nil path.
--------------------------------------------------------------------------------
local hash_t = {}
for i = 1000, 2000 do
    hash_t[i] = i
end
for i = 1000, 2000, 2 do
    hash_t[i] = nil
end

for i = 1000, 2000 do
    if i % 2 == 0 then
        assert(hash_t[i] == nil, "Test 4 failed: hash key " .. i .. " should be nil")
    else
        assert(hash_t[i] == i, "Test 4 failed: hash key " .. i .. " has wrong value")
    end
end

--------------------------------------------------------------------------------
-- 5. Re-insertion after deletion
-- Verifies that keys can be re-inserted after deletion and retain correct
-- values through multiple rounds of churn.
--------------------------------------------------------------------------------
local rt = {}
for round = 1, 3 do
    for i = 1, 100 do
        rt[i] = round * 1000 + i
    end
    for i = 1, 100, 3 do
        rt[i] = nil
    end
end

-- After 3 rounds: keys divisible by 3 offset 1 (1,4,7,...) were deleted in last round,
-- others have round-3 values
for i = 1, 100 do
    if (i - 1) % 3 == 0 then
        assert(rt[i] == nil, "Test 5 failed: key " .. i .. " should be nil after last round deletion")
    else
        assert(rt[i] == 3000 + i, "Test 5 failed: key " .. i .. " should be " .. (3000 + i))
    end
end
