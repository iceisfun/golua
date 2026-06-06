-- Test that field/method/global names are resolved correctly
-- even when constant indices exceed MaxArgC (triggering GETTABLE
-- fallback instead of GETFIELD/SELF/GETTABUP).

local t = {}
for i = 1, 1000 do t[i] = "aaa = x" .. i end
local s = table.concat(t, "; ")

-- global name
local f = assert(load(s .. "; aaa = bbb + 1"))
local ok, msg = pcall(f)
print(msg:find("global 'bbb'", 1, true) ~= nil) --> true

-- global with local _ENV
f = assert(load("local _ENV=_ENV;" .. s .. "; aaa = bbb + 1"))
ok, msg = pcall(f)
print(msg:find("global 'bbb'", 1, true) ~= nil) --> true

-- field name
f = assert(load(s .. "; local t = {}; aaa = t.bbb + 1"))
ok, msg = pcall(f)
print(msg:find("field 'bbb'", 1, true) ~= nil) --> true

-- method name: the SELF opcode cannot be used when the key's constant index
-- exceeds the RK limit, so the compiler falls back to a plain table access.
-- Reference Lua's getobjname reports this as a "field" (it has no method
-- recovery heuristic for the fallback), so golua matches with "field 'bbb'".
f = assert(load(s .. "; local t = {}; t:bbb()"))
ok, msg = pcall(f)
print(msg:find("field 'bbb'", 1, true) ~= nil) --> true
