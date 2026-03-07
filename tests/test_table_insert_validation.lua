-- Bug: table.insert doesn't validate argument count.
-- Lua 5.4 only accepts 2 or 3 arguments (table + value, or table + pos + value).

-- table.insert with 1 arg (just the table, no value) should error
local ok1, err1 = pcall(table.insert, {})
assert(not ok1, "table.insert({}) should error")
assert(tostring(err1):find("wrong number of arguments"),
  "expected 'wrong number of arguments', got: " .. tostring(err1))

-- table.insert with 4+ args should error
local ok2, err2 = pcall(table.insert, {}, 1, 2, 3)
assert(not ok2, "table.insert with 4 args should error")
assert(tostring(err2):find("wrong number of arguments"),
  "expected 'wrong number of arguments', got: " .. tostring(err2))

-- Normal 2-arg and 3-arg should still work
local t = {}
table.insert(t, "a")        -- 2 args: append
table.insert(t, 1, "b")     -- 3 args: insert at position
assert(t[1] == "b" and t[2] == "a", "normal insert should work")

