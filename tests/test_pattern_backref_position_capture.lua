-- Bug: backreference to position capture errors instead of not matching
-- In Lua 5.4, ()%1 is valid syntax; the backreference to a position capture
-- simply never matches (since position captures return numbers, not strings).
-- GoLua errors with "invalid capture index %1".

-- string.match
local ok1, r1 = pcall(string.match, "abc", "()%1")
assert(ok1, "match ()%1 should not error: " .. tostring(r1))
assert(r1 == nil, "match ()%1 should return nil")

-- string.find
local ok2, r2 = pcall(string.find, "abc", "()%1")
assert(ok2, "find ()%1 should not error: " .. tostring(r2))
assert(r2 == nil, "find ()%1 should return nil")

-- string.gsub
local ok3, r3, n3 = pcall(string.gsub, "abc", "()%1", "X")
assert(ok3, "gsub ()%1 should not error: " .. tostring(r3))
assert(r3 == "abc", "gsub ()%1 should not replace anything, got: " .. tostring(r3))
assert(n3 == 0, "gsub ()%1 should have 0 replacements")

-- string.gmatch
local ok4, r4 = pcall(function()
    local count = 0
    for w in string.gmatch("abc", "()%1") do
        count = count + 1
        if count > 5 then break end
    end
    return count
end)
assert(ok4, "gmatch ()%1 should not error: " .. tostring(r4))
assert(r4 == 0, "gmatch ()%1 should iterate 0 times")

print("PASSED")
