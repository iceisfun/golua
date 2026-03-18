-- Count hook during table constructor with function call must not clobber registers
local debug = require "debug"
local function f() return 1 end
debug.sethook(function() end, "", 1)
local t = {f()}
debug.sethook()
print(t[1])          --> 1
print(type(debug))   --> table
print(type(t))       --> table
