-- Test that constant string names appear in error messages
local ok, err

-- Attempt to call a string constant
ok, err = pcall(function() return ("hello")() end)
assert(not ok)
assert(err:find("%(constant 'hello'%)"), "expected (constant 'hello') in: " .. err)

-- Attempt bitwise on string constant
ok, err = pcall(function() return "10" ~ 1 end)
assert(not ok)
assert(err:find("%(constant '10'%)"), "expected (constant '10') in: " .. err)

-- Attempt to call a string constant in different position
ok, err = pcall(function() local x = ("world")() end)
assert(not ok)
assert(err:find("%(constant 'world'%)"), "expected (constant 'world') in: " .. err)

-- Number constants should NOT get "constant" annotation
ok, err = pcall(function() return (42)() end)
assert(not ok)
assert(not err:find("%(constant"), "number should not get (constant) annotation in: " .. err)

-- Boolean constants should NOT get "constant" annotation
ok, err = pcall(function() return (true)() end)
assert(not ok)
assert(not err:find("%(constant"), "boolean should not get (constant) annotation in: " .. err)
