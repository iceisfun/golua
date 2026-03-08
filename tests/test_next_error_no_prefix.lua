-- Test: next() error has no file:line prefix

local ok, err = pcall(function() next({a=1}, "z") end)
assert(ok == false, "expected error")
assert(err == "invalid key to 'next'",
    "expected 'invalid key to 'next'', got: " .. tostring(err))

print("OK")
