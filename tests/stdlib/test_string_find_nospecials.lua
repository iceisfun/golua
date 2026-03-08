-- Test: string.find uses plain substring search when pattern has no special characters
-- Lua 5.4 specials: ^$*+?.([%- (note: ) is NOT a special)
-- When pattern contains only non-special chars (including literal ')'),
-- string.find should do a plain substring search.

-- ) alone should match literally
assert(string.find("a)b", ")") == 2, "find ) literal")
local a, b = string.find("a)b", ")")
assert(a == 2 and b == 2, "find ) positions")

-- ) with other non-special chars
local a2, b2 = string.find("a)b", "a)b")
assert(a2 == 1 and b2 == 3, "find a)b literal")

-- ) in a string without it should return nil
assert(string.find("abc", ")") == nil, "find ) not present")

-- Multiple )
local a3, b3 = string.find("))", "))")
assert(a3 == 1 and b3 == 2, "find )) literal")

-- ) combined with a special char should use pattern mode and error
local ok, err = pcall(string.find, "a)b", ").")
assert(not ok and err:match("invalid pattern capture"), "). should error in pattern mode")

-- ] alone should also work (not a special opener)
local a4, b4 = string.find("a]b", "]")
assert(a4 == 2 and b4 == 2, "find ] literal")

-- Plain non-special pattern (no ) involved, just verifying fast path)
local a5, b5 = string.find("hello world", "world")
assert(a5 == 7 and b5 == 11, "find world")

-- Verify specials still trigger pattern matching
assert(string.find("abc", ".") == 1, ". is special")
assert(string.find("abc", "^a") == 1, "^ is special")
assert(string.find("abc", "c$") == 3, "$ is special")

print("OK")
