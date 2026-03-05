-- return;; should be a syntax error
local f, err = load("return;;")
assert(f == nil, "return;; should not compile")
assert(string.find(err, "expected"), "should mention expected: " .. tostring(err))

-- return; (single semicolon) should still be valid
local f2 = load("return;")
assert(f2 ~= nil, "return; should compile")

-- return (no semicolon) should be valid
local f3 = load("return")
assert(f3 ~= nil, "return should compile")

-- return 1; should be valid
local f4 = load("return 1;")
assert(f4 ~= nil, "return 1; should compile")

-- return 1;; should be invalid
local f5, err5 = load("return 1;;")
assert(f5 == nil, "return 1;; should not compile")
