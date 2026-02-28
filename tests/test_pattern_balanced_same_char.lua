-- BUG: %b pattern fails when open and close characters are the same
-- In Lua 5.4, %bXX (where open == close) matches from the first X
-- to the next X. GoLua always returns nil for same-char balanced matches.

-- Basic same-char balanced match
local r1 = string.match("aXa", "%baa")
assert(r1 == "aXa", "%%baa should match 'aXa', got: " .. tostring(r1))

-- Longer string
local r2 = string.match("a123a", "%baa")
assert(r2 == "a123a", "%%baa should match 'a123a', got: " .. tostring(r2))

-- Two adjacent same chars
local r3 = string.match("aa", "%baa")
assert(r3 == "aa", "%%baa should match 'aa', got: " .. tostring(r3))

-- Three same chars: matches first pair
local r4 = string.match("aaa", "%baa")
assert(r4 == "aa", "%%baa on 'aaa' should match 'aa', got: " .. tostring(r4))

-- Multiple candidates: matches first balanced pair
local r5 = string.match("aXaYa", "%baa")
assert(r5 == "aXa", "%%baa on 'aXaYa' should match 'aXa', got: " .. tostring(r5))

-- Single char: no close available
local r6 = string.match("a", "%baa")
assert(r6 == nil, "%%baa on single 'a' should return nil")

-- Same-char balanced not at start
local r7 = string.match("xxaXa", "%baa")
assert(r7 == "aXa", "%%baa should find 'aXa' in 'xxaXa', got: " .. tostring(r7))

-- Normal (different char) balanced still works
local r8 = string.match("(hello)", "%b()")
assert(r8 == "(hello)", "%%b() should still work, got: " .. tostring(r8))

-- find with same-char balanced
local s9, e9 = string.find("xxaXaxx", "%baa")
assert(s9 == 3, "find %%baa should start at 3, got: " .. tostring(s9))
assert(e9 == 5, "find %%baa should end at 5, got: " .. tostring(e9))

-- gmatch with same-char balanced
local matches = {}
for m in string.gmatch("aXa aYa", "%baa") do
    matches[#matches+1] = m
end
assert(#matches == 2, "gmatch should find 2 matches, got: " .. #matches)
assert(matches[1] == "aXa", "first match should be 'aXa', got: " .. tostring(matches[1]))
assert(matches[2] == "aYa", "second match should be 'aYa', got: " .. tostring(matches[2]))
