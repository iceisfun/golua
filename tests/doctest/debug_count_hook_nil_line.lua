-- Count hooks receive nil as the second argument in Lua 5.4, unlike line
-- hooks which receive a source line number.

debug.sethook(function(ev, line)
    print(ev, line == nil and "nil" or line)
    debug.sethook()
end, "", 1)

local x = 1
print(x)
--> =count	nil
--> =1
