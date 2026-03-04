-- Test: Native endianness for floats
-- From: tpack.lua
-- What: Tests that the default (native) float packing matches either little-endian or big-endian depending on the platform.

do
local pack = string.pack
local little = (pack("i2", 1) == "\1\0")

if little then
  assert(pack("f", 24) == pack("<f", 24))
else
  assert(pack("f", 24) == pack(">f", 24))
end
end
