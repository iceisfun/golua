-- Test: string.gsub with missing 3rd arg says "got no value"

local ok, err = pcall(string.gsub, "hello", "l")
assert(not ok, "expected error")
assert(string.find(err, "got no value", 1, true),
       "expected 'got no value' in: " .. tostring(err))

-- With explicit nil, should say "got nil"
local ok2, err2 = pcall(string.gsub, "hello", "l", nil)
assert(not ok2, "expected error")
assert(string.find(err2, "got nil", 1, true),
       "expected 'got nil' in: " .. tostring(err2))

print("OK")
