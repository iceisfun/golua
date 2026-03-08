-- Test that string.dump() with no args says "got no value"

local ok, err = pcall(string.dump)
assert(not ok)
assert(err:find("got no value"), "dump() should say 'got no value', got: " .. tostring(err))

-- With explicit nil, should say "got nil"
ok, err = pcall(string.dump, nil)
assert(not ok)
assert(err:find("got nil"), "dump(nil) should say 'got nil', got: " .. tostring(err))

-- With wrong type should say "got TYPE"
ok, err = pcall(string.dump, 42)
assert(not ok)
assert(err:find("got number"), "dump(42) should say 'got number', got: " .. tostring(err))

ok, err = pcall(string.dump, "hello")
assert(not ok)
assert(err:find("got string"), "dump('hello') should say 'got string', got: " .. tostring(err))

print("OK")
