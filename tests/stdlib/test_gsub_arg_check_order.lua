-- Test: string.gsub argument check order
-- What: reference str_gsub reads the max count (arg 4) before checking the
-- replacement type (arg 3), so when both are bad the arg 4 error comes first

local ok, err = pcall(string.gsub, "a", "a", nil, "x")
assert(not ok and err:find("bad argument #4", 1, true), err)
assert(err:find("number expected, got string", 1, true), err)

ok, err = pcall(string.gsub, "a", "a", true, 1.5)
assert(not ok and err:find("bad argument #4", 1, true), err)
assert(err:find("number has no integer representation", 1, true), err)

-- each one alone still reports its own argument
ok, err = pcall(string.gsub, "a", "a", nil)
assert(not ok and err:find("bad argument #3", 1, true), err)
ok, err = pcall(string.gsub, "a", "a", "b", "x")
assert(not ok and err:find("bad argument #4", 1, true), err)
