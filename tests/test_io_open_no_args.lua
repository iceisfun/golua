-- io.open() with no arguments should say "got no value" not "got nil".
local ok, err = pcall(io.open)
assert(not ok)
assert(tostring(err):find("got no value"),
    "io.open() should say 'got no value', got: " .. tostring(err))
