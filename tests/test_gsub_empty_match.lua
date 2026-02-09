-- gsub with empty-matching pattern should advance correctly
-- Lua 5.4 semantics: lastMatch check prevents duplicate empty match
-- "abc" with pattern "b*" gives 3 replacements: "" before a, "b" at b, "" at end
local r, n = string.gsub("abc", "b*", "Z")
assert(r == "ZaZcZ" and n == 3, "gsub empty match: expected 'ZaZcZ' n=3, got '" .. r .. "' n=" .. n)
