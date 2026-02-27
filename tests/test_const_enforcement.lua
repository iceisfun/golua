-- Bug: <const> attribute is parsed but not enforced.
-- Assigning to a const variable should be a compile-time error.

-- Test 1: direct assignment to const should fail to compile
local f1, err1 = load("local x <const> = 42; x = 43")
assert(f1 == nil, "assignment to const should fail compilation")
assert(err1:find("const"), "error should mention const: " .. tostring(err1))

-- Test 2: assignment in closure should fail
local f2, err2 = load("local x <const> = 42; local f = function() x = 43 end")
assert(f2 == nil, "assignment to const in closure should fail compilation")
assert(err2:find("const"), "error should mention const: " .. tostring(err2))

-- Test 3: reading const should work fine
local f3, err3 = load("local x <const> = 42; return x")
assert(f3, "reading const should compile: " .. tostring(err3))
assert(f3() == 42, "const value should be 42")

-- Test 4: const with table value (table itself is const, not contents)
local f4, err4 = load("local t <const> = {}; t = {}")
assert(f4 == nil, "reassignment of const table should fail")

print("PASS")
