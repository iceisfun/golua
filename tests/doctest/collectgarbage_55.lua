-- Lua 5.5: collectgarbage changes

-- "param" option reads GC parameters
print(collectgarbage("param", "pause"))
--> =250

print(collectgarbage("param", "minormul"))
--> =20

print(collectgarbage("param", "majorminor"))
--> =50

print(collectgarbage("param", "minormajor"))
--> =68

print(collectgarbage("param", "stepmul"))
--> =200

print(collectgarbage("param", "stepsize"))
--> =9600

-- "param" with a new value returns the old value
print(collectgarbage("param", "pause", 100))
--> =250

-- "setpause" is removed in 5.5
local ok, msg = pcall(collectgarbage, "setpause", 200)
print(ok)
--> =false
print(msg)
--> ~invalid option 'setpause'

-- "setstepmul" is removed in 5.5
local ok2, msg2 = pcall(collectgarbage, "setstepmul", 200)
print(ok2)
--> =false
print(msg2)
--> ~invalid option 'setstepmul'

-- invalid param name errors
local ok3, msg3 = pcall(collectgarbage, "param", "invalid")
print(ok3)
--> =false
print(msg3)
--> ~invalid option 'invalid'

-- missing param name errors
local ok4, msg4 = pcall(collectgarbage, "param")
print(ok4)
--> =false
print(msg4)
--> ~string expected, got no value
