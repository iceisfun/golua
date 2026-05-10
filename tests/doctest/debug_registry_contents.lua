-- debug.getregistry() layout in Lua 5.5: reg[1]=false (reserved),
-- reg[2]=_G (LUA_RIDX_GLOBALS), reg[3]=main thread (LUA_RIDX_MAINTHREAD,
-- moved from 1 in 5.5). Verified against lua5.5.0.

local reg = debug.getregistry()

-- reg[1] is reserved in 5.5 (was the main thread in 5.4).
print(reg[1])
--> =false

-- reg[2] should be _G (the global environment table).
print(type(reg[2]))
--> =table

print(reg[2] == _G)
--> =true

-- reg[3] should be the main coroutine thread (5.5).
print(type(reg[3]))
--> =thread
