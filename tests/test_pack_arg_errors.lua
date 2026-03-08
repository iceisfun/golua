-- Test: string.pack argument error messages

-- Missing arg should say "got nil" (not "got no value")
local ok, err = pcall(string.pack, "i4")
assert(not ok)
assert(string.find(err, "got nil"), "expected 'got nil', got: " .. tostring(err))

-- Float string should say "number has no integer representation"
local ok2, err2 = pcall(string.pack, "i4", "3.14")
assert(not ok2)
assert(string.find(err2, "no integer representation"),
    "expected 'no integer representation', got: " .. tostring(err2))

-- Integer string should work (coerced to number)
local ok3, result = pcall(string.pack, "i4", "42")
assert(ok3, "integer string should work, got: " .. tostring(result))

-- Integer value should work normally
local ok4 = pcall(string.pack, "i4", 42)
assert(ok4, "integer should work")

print("OK")
