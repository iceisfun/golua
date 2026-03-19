-- debug.upvaluejoin only accepts Lua closures on both sides. Native/C
-- closures should be rejected on the function argument itself, not on the
-- upvalue index argument.

local x = 1
local f = function()
    return x
end
local it = string.gmatch("ab", ".")

local ok1, err1 = pcall(function()
    debug.upvaluejoin(it, 1, f, 1)
end)
print(ok1, err1:find("bad argument #1", 1, true) ~= nil, err1:find("Lua function expected", 1, true) ~= nil)
--> =false	true	true

local ok2, err2 = pcall(function()
    debug.upvaluejoin(f, 1, it, 1)
end)
print(ok2, err2:find("bad argument #3", 1, true) ~= nil, err2:find("Lua function expected", 1, true) ~= nil)
--> =false	true	true
