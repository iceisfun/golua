-- Test: multiple <close> attributes in a single local statement are rejected
-- Lua 5.4 allows at most one to-be-closed variable per local statement.

-- Multiple <close> variables should fail
local ok, err = load("local x <close>, y <close> = nil, nil")
assert(not ok, "multiple <close> should be rejected")
assert(string.find(err, "multiple to%-be%-closed"), "error should mention multiple to-be-closed")

-- Single <close> should work
local ok2 = load("local x <close> = nil")
assert(ok2, "single <close> should be accepted")

-- <close> with other non-close variable should work
local ok3 = load("local x <close>, y = nil, nil")
assert(ok3, "<close> + plain should be accepted")

-- <close> on non-first variable should work
local ok4 = load("local x, y <close> = nil, nil")
assert(ok4, "plain + <close> should be accepted")

-- <const> + <close> should work (different attributes)
local ok5 = load("local x <const>, y <close> = 1, nil")
assert(ok5, "<const> + <close> should be accepted")

-- Three variables with two <close> should fail
local ok6, err6 = load("local a <close>, b, c <close> = nil, nil, nil")
assert(not ok6, "two <close> in three vars should be rejected")

