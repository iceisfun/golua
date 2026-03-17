local ok, err
ok, err = pcall(string.pack, "b", 999)
assert(string.find(err, "'string%.pack'"), "expected 'string.pack', got: " .. tostring(err))

ok, err = pcall(string.unpack, "z", "no null")
assert(string.find(err, "'string%.unpack'"), "expected 'string.unpack', got: " .. tostring(err))

ok, err = pcall(string.packsize, "z")
assert(string.find(err, "'string%.packsize'"), "expected 'string.packsize', got: " .. tostring(err))
print("OK")
