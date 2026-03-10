-- Test: string.unpack error message formats

-- Unfinished string for 'z' should have proper prefix
local ok, err = pcall(string.unpack, "z", "no null")
assert(not ok)
assert(string.find(err, "bad argument #2 to 'unpack'"),
    "expected 'bad argument #2' prefix, got: " .. tostring(err))
assert(string.find(err, "unfinished string for format 'z'"),
    "expected 'unfinished string' message, got: " .. tostring(err))

-- Position out of string should say "initial position"
local ok2, err2 = pcall(string.unpack, "i4", "abc", 10)
assert(not ok2)
assert(string.find(err2, "initial position out of string"),
    "expected 'initial position out of string', got: " .. tostring(err2))

print("OK")
