-- Frontier pattern (%f[]) tests
-- Tracking: Phase 2 (pattern engine completion)
--
-- The frontier pattern %f[set] matches at a position where the character
-- before does not match [set] but the character after does. This is used
-- for word boundary matching and similar tasks.

-- At start of string, prev is treated as \0 (not alpha), so %f[%a] matches pos 1
local r = string.find("THE END", "%f[%a]%u+")
assert(r == 1, "frontier at start of string should match 'THE' at position 1, got: " .. tostring(r))

-- Find second word boundary: use %f to find word starting after a space
local r2 = string.find("THE END", "%f[%a]%u+", 2)
assert(r2 == 5, "frontier pattern should match 'END' at position 5, got: " .. tostring(r2))

-- Frontier with non-alpha boundary: find start of word after non-alpha
local s = "hello world"
local r3 = string.find(s, "%f[%a]world")
assert(r3 == 7, "frontier should find 'world' at position 7, got: " .. tostring(r3))

-- Frontier at end: %f[%A] matches transition from alpha to non-alpha
local r4, r5 = string.find("abc def", "%a+%f[%A]")
assert(r4 == 1 and r5 == 3, "should match 'abc' ending at word boundary")

-- Frontier should not match when there is no boundary
local r6 = string.find("abcdef", "%f[%a]", 3)
assert(r6 == nil, "no frontier in middle of word")
