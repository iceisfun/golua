-- string.rep with huge count should error gracefully, not crash
local ok, err = pcall(string.rep, "x", 2^30)
assert(not ok, "expected error for huge string.rep")
-- The error should be catchable (we got here, so it was)
print("OK")
