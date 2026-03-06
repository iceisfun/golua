-- yield-across-boundary error should NOT have file:line prefix
local co = coroutine.create(function()
    table.sort({1,2}, function(a,b) coroutine.yield() return a<b end)
end)
local ok, err = coroutine.resume(co)
assert(not ok)
assert(err == "attempt to yield across a C-call boundary", "got: " .. tostring(err))

print("PASS")
