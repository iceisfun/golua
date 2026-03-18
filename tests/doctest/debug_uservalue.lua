-- debug.getuservalue and debug.setuservalue behavior tests

-- getuservalue on non-userdata returns nil (1 value)
print(select("#", debug.getuservalue({})))
--> =1
print(debug.getuservalue({}))
--> =nil
print(select("#", debug.getuservalue(42)))
--> =1

-- setuservalue on non-userdata errors with "full userdata expected"
local ok, err = pcall(debug.setuservalue, {}, 10)
print(ok)
--> =false
print(err:find("full userdata expected, got table") ~= nil)
--> =true

-- setuservalue on light userdata errors with "light userdata"
local x = 1
local function f() return x end
local id = debug.upvalueid(f, 1)
local ok2, err2 = pcall(debug.setuservalue, id, 10)
print(ok2)
--> =false
print(err2:find("full userdata expected, got light userdata") ~= nil)
--> =true
