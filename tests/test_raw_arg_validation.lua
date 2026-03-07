-- Verify rawget, rawset, rawequal validate argument counts

-- rawget needs 2 args
local ok1, err1 = pcall(rawget, {})
assert(not ok1, "rawget({}) should fail")
assert(err1:find("bad argument #2"), "rawget 1 arg: " .. tostring(err1))

-- rawset needs 3 args
local ok2, err2 = pcall(rawset, {})
assert(not ok2, "rawset({}) should fail")
assert(err2:find("bad argument #2"), "rawset 1 arg: " .. tostring(err2))

local ok3, err3 = pcall(rawset, {}, "x")
assert(not ok3, "rawset({}, 'x') should fail")
assert(err3:find("bad argument #3"), "rawset 2 args: " .. tostring(err3))

-- rawequal needs 2 args
local ok4, err4 = pcall(rawequal, 1)
assert(not ok4, "rawequal(1) should fail")
assert(err4:find("bad argument #2"), "rawequal 1 arg: " .. tostring(err4))

-- Normal operation still works
local t = {}
rawset(t, "x", 42)
assert(rawget(t, "x") == 42)
assert(rawequal(1, 1) == true)
assert(rawequal(1, 2) == false)

