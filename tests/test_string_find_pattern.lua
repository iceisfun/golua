-- test_string_find_pattern.lua
-- string.find should interpret Lua patterns, return captures, and compute boundaries.

local text = "abc123def456"

local start_pos, end_pos = string.find(text, "%d+")
assert(start_pos == 4 and end_pos == 6, string.format("expected match 4..6, got %s..%s", tostring(start_pos), tostring(end_pos)))

local s2, e2, digits = string.find(text, "(%d+)")
assert(digits == "123", string.format("expected capture '123', got %s", tostring(digits)))
assert(s2 == 4 and e2 == 6, string.format("expected capture bounds 4..6, got %s..%s", tostring(s2), tostring(e2)))

-- When plain flag is set, the pattern should be treated as a literal
local plain_start = select(1, string.find(text, "%d+", 1, true))
assert(plain_start == nil, "plain search should treat % as literal and fail to match")
