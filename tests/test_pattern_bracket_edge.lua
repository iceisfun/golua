-- Bug 1: ] at start of character set not treated as literal.
-- In Lua 5.4, []] matches a literal ], and []abc] matches ], a, b, or c.

-- []] should match a literal ]
assert(string.find("]", "[]]"),
  "[]] should match ']'")

-- []abc] should match ], a, b, or c
assert(string.find("]", "[]abc]"),
  "[]abc] should match ']'")
assert(string.find("a", "[]abc]"),
  "[]abc] should match 'a'")

-- Bug 2: Malformed [ pattern should error, not silently return no match
local ok, err = pcall(string.find, "hello", "[")
assert(not ok, "malformed '[' pattern should error, not return no match")
assert(tostring(err):find("malformed"),
  "error should mention 'malformed', got: " .. tostring(err))

print("PASS")
