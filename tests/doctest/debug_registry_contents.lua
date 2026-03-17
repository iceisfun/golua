-- debug.getregistry() should contain the main thread at [1] and _G at [2].
-- These are standard entries that C Lua always populates.

local reg = debug.getregistry()

-- reg[1] should be the main coroutine thread
print(type(reg[1]))
--> =thread

-- reg[2] should be _G (the global environment table)
print(type(reg[2]))
--> =table

print(reg[2] == _G)
--> =true
