-- Test: table.sort must detect invalid order function
-- Lua 5.4 detects when a comparator is inconsistent (e.g., always returns true)
-- and raises "invalid order function for sorting". GoLua silently succeeds.

-- A comparator that always returns true violates the strict weak ordering contract.
-- Lua 5.4 detects this during sorting and errors.
local t = {3, 1, 4, 1, 5, 9, 2, 6}
local ok, err = pcall(table.sort, t, function(a, b) return true end)
assert(not ok, "table.sort should detect always-true comparator")
assert(string.find(tostring(err), "invalid order function"),
  "expected 'invalid order function' error, got: " .. tostring(err))
