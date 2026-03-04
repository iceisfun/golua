-- Test: Minimum integer format behavior
-- From: tpack.lua
-- What: Tests pack/unpack round-tripping for minimum-size integer formats (B/b, H/h, L/l) at their boundary values.

do
local pack = string.pack
local unpack = string.unpack

assert(unpack("B", pack("B", 0xff)) == 0xff)
assert(unpack("b", pack("b", 0x7f)) == 0x7f)
assert(unpack("b", pack("b", -0x80)) == -0x80)

assert(unpack("H", pack("H", 0xffff)) == 0xffff)
assert(unpack("h", pack("h", 0x7fff)) == 0x7fff)
assert(unpack("h", pack("h", -0x8000)) == -0x8000)

assert(unpack("L", pack("L", 0xffffffff)) == 0xffffffff)
assert(unpack("l", pack("l", 0x7fffffff)) == 0x7fffffff)
assert(unpack("l", pack("l", -0x80000000)) == -0x80000000)
end
