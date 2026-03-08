-- Test: Parser "near" token reports raw byte value for high bytes (0x80-0xFF)

-- 0x80 should report <\128>, not <\194> (UTF-8 lead byte)
local ok1, err1 = load("local x = \x80")
assert(not ok1)
assert(string.find(err1, "<\\128>", 1, true),
       "expected '<\\128>' in: " .. tostring(err1))

-- 0xFF should report <\255>
local ok2, err2 = load("local x = \xFF")
assert(not ok2)
assert(string.find(err2, "<\\255>", 1, true),
       "expected '<\\255>' in: " .. tostring(err2))

-- 0xA0 should report <\160>
local ok3, err3 = load("local x = \xA0")
assert(not ok3)
assert(string.find(err3, "<\\160>", 1, true),
       "expected '<\\160>' in: " .. tostring(err3))

print("OK")
