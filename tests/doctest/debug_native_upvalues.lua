-- C closure nups and getupvalue for native functions
local iter = string.gmatch("abc", "a")
print(debug.getinfo(iter).nups)        --> 3

-- getupvalue returns "" as name for C closure upvalues
local name, val = debug.getupvalue(iter, 1)
print(type(name))                      --> string
print(#name)                           --> 0
print(val)                             --> abc

local name2 = debug.getupvalue(iter, 2)
print(type(name2))                     --> string
print(#name2)                          --> 0

-- out of range returns no values
print(debug.getupvalue(iter, 0) == nil) --> true
print(debug.getupvalue(iter, 4) == nil) --> true

-- coroutine.wrap reports 1 upvalue
local co = coroutine.wrap(function() end)
print(debug.getinfo(co).nups)          --> 1
local cname, cval = debug.getupvalue(co, 1)
print(type(cname))                     --> string
print(#cname)                          --> 0
print(type(cval))                      --> thread

-- setupvalue on native functions
local sname = debug.setupvalue(iter, 1, "xyz")
print(type(sname))                     --> string
print(#sname)                          --> 0
print(debug.setupvalue(iter, 5, "z") == nil) --> true
