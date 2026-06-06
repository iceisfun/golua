-- debug.getuservalue and debug.setuservalue behavior tests

-- getuservalue on non-userdata returns nil (1 value)
print(select("#", debug.getuservalue({})))
--> =1
print(debug.getuservalue({}))
--> =nil
print(select("#", debug.getuservalue(42)))
--> =1

-- setuservalue on non-userdata errors with "userdata expected" (matches lua5.5.0)
local ok, err = pcall(debug.setuservalue, {}, 10)
print(ok)
--> =false
print(err:find("userdata expected, got table") ~= nil)
--> =true

-- setuservalue on light userdata: reference's luaL_typeerror distinguishes
-- light userdata (e.g. debug.upvalueid results) from full userdata, so the
-- "got" type reads "light userdata" (verified against lua5.5.0).
local x = 1
local function f() return x end
local id = debug.upvalueid(f, 1)
local ok2, err2 = pcall(debug.setuservalue, id, 10)
print(ok2)
--> =false
print(err2:find("userdata expected, got light userdata") ~= nil)
--> =true
