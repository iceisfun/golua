-- Test: Variable-width integer formats (i1..i16)
-- From: tpack.lua
-- What: Tests pack/unpack with variable-width integer formats from 1 to 16 bytes, verifying sign extension for -1, small unsigned values, and both little/big endian.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack
local NB = 16

for i = 1, NB do
  -- small numbers with signal extension ("\xFF...")
  local s = string.rep("\xff", i)
  assert(pack("i" .. i, -1) == s)
  assert(packsize("i" .. i) == #s)
  assert(unpack("i" .. i, s) == -1)

  -- small unsigned number ("\0...\xAA")
  s = "\xAA" .. string.rep("\0", i - 1)
  assert(pack("<I" .. i, 0xAA) == s)
  assert(unpack("<I" .. i, s) == 0xAA)
  assert(pack(">I" .. i, 0xAA) == s:reverse())
  assert(unpack(">I" .. i, s:reverse()) == 0xAA)
end
end
