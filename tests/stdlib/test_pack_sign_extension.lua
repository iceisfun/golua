-- Test: Sign extension
-- From: tpack.lua
-- What: Tests sign extension behavior when unpacking signed vs unsigned integers with leading 0xF0 bytes.

do
local packsize = string.packsize
local unpack = string.unpack
local sizeLI = packsize("j")

do
  local u = 0xf0
  for i = 1, sizeLI - 1 do
    assert(unpack("<i"..i, "\xf0"..("\xff"):rep(i - 1)) == -16)
    assert(unpack(">I"..i, "\xf0"..("\xff"):rep(i - 1)) == u)
    u = u * 256 + 0xff
  end
end
end
