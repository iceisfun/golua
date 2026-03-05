-- Verify select() error messages match Lua 5.4 semantics

-- Float that can't be integer: "number has no integer representation"
local ok, err = pcall(select, 1.5, 10, 20, 30)
assert(not ok)
assert(err:find("no integer representation"), "select(1.5) error: " .. tostring(err))

-- Non-number string: "number expected, got string"
local ok2, err2 = pcall(select, "abc", 10, 20, 30)
assert(not ok2)
assert(err2:find("number expected"), "select('abc') error: " .. tostring(err2))
assert(err2:find("got string"), "select('abc') missing 'got string': " .. tostring(err2))

-- Boolean: "number expected, got boolean"
local ok3, err3 = pcall(select, true, 10, 20, 30)
assert(not ok3)
assert(err3:find("number expected"), "select(true) error: " .. tostring(err3))
assert(err3:find("got boolean"), "select(true) missing 'got boolean': " .. tostring(err3))

print("PASSED")
