-- table.concat should error when range includes non-string/non-number elements.
-- Lua 5.4: "invalid value (nil) at index N in table for 'concat'"

-- Explicit range that includes nil should error
local ok1, e1 = pcall(table.concat, {"a", nil, "b"}, ",", 1, 3)
assert(not ok1, "table.concat({'a',nil,'b'}, ',', 1, 3) should error")

-- Boolean should also error
local ok2, e2 = pcall(table.concat, {true}, ",", 1, 1)
assert(not ok2, "table.concat({true}, ',', 1, 1) should error")

-- Table should also error
local ok3, e3 = pcall(table.concat, {{}}, ",", 1, 1)
assert(not ok3, "table.concat({{}}, ',', 1, 1) should error")

-- Guard: tables without nil should keep working
assert(table.concat({1, 2, 3}, ",") == "1,2,3")
assert(table.concat({"a", "b"}) == "ab")
assert(table.concat({}) == "")

-- Guard: explicit range that avoids nil should work
assert(table.concat({"a", nil, "c"}, ",", 1, 1) == "a")
