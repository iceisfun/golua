-- Test: Integer packing with exact bit widths
-- From: tpack.lua
-- What: Tests packing integers into exact byte widths from 1 to sizeLI, verifying both little and big endian byte ordering.
-- Broken: GoLua lexer converts oversized hex literals (>64 bits) to float instead of wrapping to int64

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack
local sizeLI = packsize("j")

for i = 1, sizeLI do
  local lstr = "\1\2\3\4\5\6\7\8\9\10\11\12\13"
  local lnum = 0x13121110090807060504030201
  local n = lnum & (~(-1 << (i * 8)))
  local s = string.sub(lstr, 1, i)
  assert(pack("<i" .. i, n) == s)
  assert(pack(">i" .. i, n) == s:reverse())
  assert(unpack(">i" .. i, s:reverse()) == n)
end
end
