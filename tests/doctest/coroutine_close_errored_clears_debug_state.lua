local co = coroutine.create(function(a)
    local b = 2
    error("boom")
end)

coroutine.resume(co, 1)
coroutine.close(co)

print(debug.getinfo(co, 1) == nil)
--> =true

local ok, err = pcall(debug.getlocal, co, 1, 1)
print(ok)
--> =false

print(string.find(err, "level out of range", 1, true) ~= nil)
--> =true

print(debug.traceback(co))
--> =stack traceback:
