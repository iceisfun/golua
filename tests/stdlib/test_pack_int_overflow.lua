-- Test: Integer overflow in packing
-- From: tpack.lua
-- What: Tests that packing integers outside their valid range (both signed and unsigned) produces overflow errors, and verifies boundary values pack correctly.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack
local sizeLI = packsize("j")

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

for i = 1, sizeLI - 1 do
  local umax = (1 << (i * 8)) - 1
  local max = umax >> 1
  local min = ~max
  checkerror("overflow", pack, "<I" .. i, -1)
  checkerror("overflow", pack, "<I" .. i, min)
  checkerror("overflow", pack, ">I" .. i, umax + 1)

  checkerror("overflow", pack, ">i" .. i, umax)
  checkerror("overflow", pack, ">i" .. i, max + 1)
  checkerror("overflow", pack, "<i" .. i, min - 1)

  assert(unpack(">i" .. i, pack(">i" .. i, max)) == max)
  assert(unpack("<i" .. i, pack("<i" .. i, min)) == min)
  assert(unpack(">I" .. i, pack(">I" .. i, umax)) == umax)
end
end
