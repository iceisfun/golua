-- Test: Mixed endianness
-- From: tpack.lua
-- What: Tests combining big-endian and little-endian formats in the same pack/unpack format string, and the native-endian = prefix.

do
local pack = string.pack
local unpack = string.unpack

do
  assert(pack(">i2 <i2", 10, 20) == "\0\10\20\0")
  local a, b = unpack("<i2 >i2", "\10\0\0\20")
  assert(a == 10 and b == 20)
  assert(pack("=i4", 2001) == pack("i4", 2001))
end
end
