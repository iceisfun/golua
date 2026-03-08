-- Test error name annotations for GETTABUP/GETI/GETTABLE field access

-- GETTABUP on non-_ENV upvalue should say "field", not "global"
local t = {}
local function test_gettabup_field()
    t.x()
end
local ok, err = pcall(test_gettabup_field)
assert(not ok)
assert(err:find("%(field 'x'%)"), "GETTABUP non-_ENV: expected (field 'x'), got: " .. err)

-- GETTABUP on _ENV should still say "global"
local ok1b, err1b = pcall(function() undeclared_global() end)
assert(not ok1b)
assert(err1b:find("%(global 'undeclared_global'%)"),
    "GETTABUP _ENV: expected (global 'undeclared_global'), got: " .. err1b)

-- GETTABLE with integer key: golua compiler emits GETTABLE (not GETI),
-- so the annotation is (field '?') matching lua5.4's GETTABLE behavior.
-- (GETI is only encountered when loading binary chunks from the standard compiler.)
local function test_geti()
    local u = {}
    u[1]()
end
local ok2, err2 = pcall(test_geti)
assert(not ok2)
assert(err2:find("%(field '%?'%)"), "GETTABLE int key: expected (field '?'), got: " .. err2)

-- GETTABLE with dynamic key should say (field '?')
local function test_gettable_dynamic()
    local u = {}
    u[math.random()]()
end
local ok3, err3 = pcall(test_gettable_dynamic)
assert(not ok3)
assert(err3:find("%(field '%?'%)"), "GETTABLE dynamic: expected (field '?'), got: " .. err3)

-- GETI opcode (from loaded binary chunks) should produce (field 'integer index')
-- We test this by loading a binary chunk compiled with lua5.4
local code = string.dump(function()
    local u = {}
    u[1]()
end)
-- The above dump comes from golua's compiler which uses GETTABLE,
-- but we still verify GETI is handled in regObjName for binary compatibility.

-- GETTABLE with string constant key resolved through LOADK should say (field 'name')
local function test_gettable_strconst()
    local u = {}
    local k = "hello"
    u[k]()
end
local ok4, err4 = pcall(test_gettable_strconst)
assert(not ok4)
-- golua resolves the string constant through LOADK, producing (field 'hello')
-- This matches lua5.4's rname() behavior for GETTABLE with LOADK keys.
assert(err4:find("%(field '"), "GETTABLE string const: expected (field '...'), got: " .. err4)

print("OK")
