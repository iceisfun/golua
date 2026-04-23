-- Verify debug.traceback labels frames using nameWhat kinds that match
-- lua5.5.0: "in global", "in field", "in method", "in local", "in upvalue"
-- for both Lua-frame and C-frame entries.

-- "in global '<name>'" for a Lua frame invoked via _ENV
local function boom() error("boom") end
_G.gfn = boom

local ok1, err1 = xpcall(function()
    gfn()
end, debug.traceback)
assert(not ok1)
assert(string.find(err1, "in global 'gfn'", 1, true),
    "expected \"in global 'gfn'\" (Lua frame), got:\n" .. err1)

-- "[C]: in global '<name>'" for a C function looked up via _ENV
local ok2, err2 = xpcall(function()
    error("x")
end, debug.traceback)
assert(not ok2)
assert(string.find(err2, "[C]: in global 'error'", 1, true),
    "expected \"[C]: in global 'error'\", got:\n" .. err2)

-- xpcall itself is a global; it should show up as [C]: in global 'xpcall'
assert(string.find(err2, "[C]: in global 'xpcall'", 1, true),
    "expected \"[C]: in global 'xpcall'\", got:\n" .. err2)

-- "[C]: in field '<name>'" for a C function looked up via t.f
local ok3, err3 = xpcall(function()
    table.sort(nil)
end, debug.traceback)
assert(not ok3)
assert(string.find(err3, "[C]: in field '", 1, true),
    "expected \"[C]: in field '\" for table.sort, got:\n" .. err3)

-- "in method '<name>'" for a Lua function invoked via obj:m()
local ok4, err4 = xpcall(function()
    local obj = setmetatable({}, {__index = {m = function(self) error("e") end}})
    obj:m()
end, debug.traceback)
assert(not ok4)
assert(string.find(err4, "in method 'm'", 1, true),
    "expected \"in method 'm'\", got:\n" .. err4)

-- "in local '<name>'" for a Lua function invoked via a local binding
local ok5, err5 = xpcall(function()
    local lf = function() error("e") end
    lf()
end, debug.traceback)
assert(not ok5)
assert(string.find(err5, "in local 'lf'", 1, true),
    "expected \"in local 'lf'\", got:\n" .. err5)

-- "in upvalue '<name>'" for a Lua function invoked via an upvalue
local up = function() error("e") end
local function call_upvalue()
    up()
end
local ok6, err6 = xpcall(call_upvalue, debug.traceback)
assert(not ok6)
assert(string.find(err6, "in upvalue 'up'", 1, true),
    "expected \"in upvalue 'up'\", got:\n" .. err6)

-- "in field '<name>'" for a Lua function invoked via t.f
local t = {}
t.myfield = function() error("e") end
local ok7, err7 = xpcall(function()
    t.myfield()
end, debug.traceback)
assert(not ok7)
assert(string.find(err7, "in field 'myfield'", 1, true),
    "expected \"in field 'myfield'\", got:\n" .. err7)

print("OK")
