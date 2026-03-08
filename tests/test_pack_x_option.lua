-- Test: X option accepts 's' as next format option

-- Xs should be valid in packsize
local ok, size = pcall(string.packsize, "Xs")
assert(ok, "Xs should be accepted by packsize, got: " .. tostring(size))
assert(size == 0, "Xs should have size 0, got: " .. tostring(size))

-- Xs2 should also work (s with explicit prefix size 2)
local ok2, size2 = pcall(string.packsize, "Xs2")
assert(ok2, "Xs2 should be accepted, got: " .. tostring(size2))

print("OK")
