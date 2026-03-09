-- Test traceback formatting: namewhat and anonymous function source

-- Bug 1: namewhat should be used in traceback (local/field/upvalue/method, not always "function")

-- Test "in local 'f'"
local ok2, err2 = xpcall(function()
    local f = function() error("boom") end
    f()
end, debug.traceback)
assert(not ok2)
assert(string.find(err2, "in local 'f'"), "expected 'in local' for local var, got:\n" .. err2)

-- Test "in field 'myfield'"
local t = {}
t.myfield = function() error("boom") end
local ok3, err3 = xpcall(function()
    t.myfield()
end, debug.traceback)
assert(not ok3)
assert(string.find(err3, "in field 'myfield'"), "expected 'in field' for table field, got:\n" .. err3)

-- Test "in upvalue 'up'"
local up = function() error("boom") end
local function call_upvalue()
    up()
end
local ok5, err5 = xpcall(call_upvalue, debug.traceback)
assert(not ok5)
assert(string.find(err5, "in upvalue 'up'"), "expected 'in upvalue' for upvalue, got:\n" .. err5)

-- Bug 2: anonymous function source should not have extra quotes
-- Lua 5.4: "in function <file:line>" not "in function '<file:line>'"
local ok6, err6 = xpcall(function()
    (function() error("boom") end)()
end, debug.traceback)
assert(not ok6)
-- Should have "in function <" not "in function '<"
assert(string.find(err6, "in function <"), "expected 'in function <file:line>' without quotes, got:\n" .. err6)
assert(not string.find(err6, "in function '<"), "should NOT have quotes around <file:line>, got:\n" .. err6)

print("OK")
