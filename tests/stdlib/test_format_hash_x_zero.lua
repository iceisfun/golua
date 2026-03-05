-- string.format %#x/%#X/%#o with value 0 should not produce a prefix.
-- C printf: the # flag has no effect when the value is zero.

-- %#x with 0: should be "0", not "0x0"
assert(string.format("%#x", 0) == "0", "got: " .. string.format("%#x", 0))

-- %#X with 0: should be "0", not "0X0"
assert(string.format("%#X", 0) == "0", "got: " .. string.format("%#X", 0))

-- %#o with 0: should be "0", not "00"
assert(string.format("%#o", 0) == "0", "got: " .. string.format("%#o", 0))

-- Non-zero values should still get the prefix
assert(string.format("%#x", 1) == "0x1")
assert(string.format("%#X", 1) == "0X1")
assert(string.format("%#o", 1) == "01")
assert(string.format("%#x", 255) == "0xff")
assert(string.format("%#X", 255) == "0XFF")

-- Width formatting with # and zero value
assert(string.format("%#8x", 0) == "       0", "got: [" .. string.format("%#8x", 0) .. "]")

print("OK")
