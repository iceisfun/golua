-- Test that next() terminates even when table is modified during iteration
-- This prevents an infinite loop bug where newly added keys keep extending iteration

-- Test 1: Adding keys during iteration should terminate
-- In Lua 5.4, this terminates because the hash table has fixed size.
-- New keys may or may not be visited, but iteration always terminates.
local t = {a=1, b=2, c=3}
local count = 0
for k, v in next, t do
    t[k .. "x"] = v  -- always add new key
    count = count + 1
    if count > 100 then
        break  -- safety valve
    end
end
assert(count <= 100, "next() iteration should terminate, got " .. count .. " iterations")
print("iteration terminated after " .. count .. " steps")

-- Test 2: Basic next() still works after the fix
local t2 = {x=1, y=2, z=3}
local keys = {}
for k, v in next, t2 do
    keys[#keys+1] = k
end
assert(#keys == 3, "basic next() should visit all 3 keys, got " .. #keys)

-- Test 3: Deleting keys during iteration is fine
local t3 = {a=1, b=2, c=3, d=4}
local visited = 0
for k, v in next, t3 do
    t3[k] = nil  -- delete current key
    visited = visited + 1
end
assert(visited >= 1, "should visit at least 1 key")

-- Test 4: Adding keys without delete should also terminate
local t4 = {p=1, q=2}
count = 0
for k, v in next, t4 do
    t4[k .. "y"] = v + 1
    count = count + 1
    if count > 50 then break end
end
assert(count <= 50, "add-only iteration should terminate")

print("OK")
