-- Test protected call with error
local ok, err = pcall(function()
    error("boom")
end)

assert(ok == false)
assert(string.match(err, "boom"))
