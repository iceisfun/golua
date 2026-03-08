-- Test pack alignment edge cases

-- !5 with i4 should work (effective align = min(5,4) = 4)
local s = string.pack("!5 xi4", 42)
assert(#s >= 4, "!5 i4 should work")

-- !5 with i8 should fail (effective align = min(5,8) = 5, NOT power of 2)
local ok, err = pcall(string.pack, "!5 i8", 42)
assert(not ok, "!5 i8 should fail: effective align = 5")

-- !6 with i4 should work (effective align = min(6,4) = 4)
s = string.pack("!6 xi4", 42)
assert(#s >= 4, "!6 i4 should work")

-- !7 with i4 should work (effective align = min(7,4) = 4)
s = string.pack("!7 xi4", 42)
assert(#s >= 4, "!7 i4 should work")

-- !3 i2 -> min(3,2) = 2 -> ok
ok, err = pcall(string.pack, "!3 i2", 42)
assert(ok, "!3 i2 should work: effective align min(3,2)=2: " .. tostring(err))

-- !3 i4 -> min(3,4) = 3, not pow2 -> error
ok, err = pcall(string.pack, "!3 i4", 42)
assert(not ok, "!3 i4 should fail: effective align = 3")

-- packsize should also work
assert(string.packsize("!5 i4") >= 4, "packsize !5 i4 should work")

-- unpack should also work
s = string.pack("!4 i4", 42)
local v = string.unpack("!4 i4", s)
assert(v == 42)

-- 'c' without size error message
ok, err = pcall(string.pack, "c")
assert(not ok)
assert(err:find("missing size for format option 'c'"), "c error: " .. tostring(err))

ok, err = pcall(string.packsize, "c")
assert(not ok)
assert(err:find("missing size for format option 'c'"), "c packsize error: " .. tostring(err))

print("OK")
