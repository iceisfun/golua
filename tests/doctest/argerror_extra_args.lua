-- Lua 5.5 luaL_argerror: an argument added by a __call metamethod is a
-- "bad extra argument", and the arguments the caller passed are numbered
-- without the extra ones.

local function e(f, ...)
  local ok, m = pcall(f, ...)
  return ok, (m:gsub("^[^:]*:%d+: ", ""))
end

local r = setmetatable({}, {__call = string.rep})
print(e(function() return r(3) end))
--> =false	bad extra argument #1 to 'r' (string expected, got table)
print(e(function() r(3) end))
--> =false	bad extra argument #1 to 'r' (string expected, got table)

-- called from C (pcall) the extra argument still counts
print(e(setmetatable({}, {__call = select}), '#', 1))
--> =false	bad extra argument #1 to 'select' (number expected, got table)

-- two __call levels add two extra arguments
local outer = setmetatable({}, {__call = r})
print(e(function() return outer('x') end))
--> =false	bad extra argument #1 to 'outer' (string expected, got table)

-- the caller's own arguments are numbered from 1
local sm = setmetatable({}, {__call = setmetatable})
print(e(function() sm(5) end))
--> =false	bad argument #1 to 'sm' (nil or table expected, got number)
print(e(sm, 5))
--> =false	bad argument #1 to 'setmetatable' (nil or table expected, got number)
local c = setmetatable({}, {__call = table.concat})
local c2 = setmetatable({}, {__call = c})
print(e(function() c2('x') end))
--> =false	bad extra argument #2 to 'c2' (string expected, got table)

-- a native reached by a tail call counts only the __call levels on the way
-- to it, not those of the frame making the tail call
local wt = setmetatable({}, {__call = function(self) return sm(5) end})
print(e(wt))
--> =false	bad argument #1 to 'sm' (nil or table expected, got number)

debug.setmetatable(0, {__call = string.rep})
print(e(function() local f = 5 return f(2, {}) end))
--> =false	bad argument #2 to 'f' (string expected, got table)
debug.setmetatable(0, nil)

-- string.pack numbers its own arguments the same way
debug.setmetatable("", {__call = string.pack, __index = string})
local fmt = "i4"
print(e(function() return fmt(2^40) end))
--> =false	bad argument #1 to 'fmt' (integer overflow)
debug.setmetatable("", {__index = string})
