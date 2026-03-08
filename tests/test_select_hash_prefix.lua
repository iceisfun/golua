-- Test that select() accepts any string starting with '#'

-- Standard usage
assert(select("#", 1, 2, 3) == 3)

-- Strings starting with '#' should also work
assert(select("##", 1, 2, 3) == 3)
assert(select("#anything", 1, 2) == 2)
assert(select("# ", 1, 2) == 2)

-- Non-hash strings should error
local ok, err = pcall(select, "abc")
assert(not ok)
assert(string.find(err, "number expected, got string"), err)

print("OK")
