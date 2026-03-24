local ok, err = pcall(debug.getinfo, 99, ">")
print(ok)
--> =false

print(string.find(err, "invalid option '>'", 1, true) ~= nil)
--> =true

local co = coroutine.create(function()
    coroutine.yield()
end)
coroutine.resume(co)

local ok2, err2 = pcall(debug.getinfo, co, 99, ">")
print(ok2)
--> =false

print(string.find(err2, "invalid option '>'", 1, true) ~= nil)
--> =true
