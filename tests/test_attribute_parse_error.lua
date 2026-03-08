-- Test that attribute parsing gives correct error messages

-- Empty attribute
local ok, err = load("local x <> = 1")
assert(not ok, "empty attribute should fail")
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- Number as attribute
ok, err = load("local x <42> = 1")
assert(not ok, "number attribute should fail")
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- Keyword as attribute
ok, err = load("local x <end> = 1")
assert(not ok, "keyword attribute should fail")
-- 'end' is a keyword not a name, Lua 5.4 gives <name> expected
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- Boolean as attribute
ok, err = load("local x <true> = 1")
assert(not ok, "boolean attribute should fail")
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- Nil as attribute
ok, err = load("local x <nil> = 1")
assert(not ok, "nil attribute should fail")
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- String as attribute
ok, err = load('local x <"const"> = 1')
assert(not ok, "string attribute should fail")
assert(err:find("<name> expected"), "expected '<name> expected' in: " .. err)

-- Valid attributes should still work
local f1 = load("local x <const> = 1")
assert(f1, "const attribute should work")

local f2 = load("local x <close> = setmetatable({}, {__close=function() end})")
assert(f2, "close attribute should work")

-- Invalid but name-shaped attribute
ok, err = load("local x <frozen> = 1")
assert(not ok, "unknown attribute should fail")
assert(err:find("unknown attribute"), "expected 'unknown attribute' in: " .. err)

print("OK")
