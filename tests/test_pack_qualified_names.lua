local ok, err
ok, err = pcall(string.pack, "b", 999)
assert(string.find(err, "'pack'"), "expected 'pack', got: " .. tostring(err))

ok, err = pcall(string.unpack, "z", "no null")
assert(string.find(err, "'unpack'"), "expected 'unpack', got: " .. tostring(err))

ok, err = pcall(string.packsize, "z")
assert(string.find(err, "'packsize'"), "expected 'packsize', got: " .. tostring(err))
print("OK")
