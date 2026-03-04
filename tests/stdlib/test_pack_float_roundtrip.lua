-- Test: Floating-point number pack/unpack
-- From: tpack.lua
-- What: Tests pack/unpack round-tripping for float and double types in native, little-endian, and big-endian formats, including special values (0, inf, negative).

do
local pack = string.pack
local unpack = string.unpack

for _, n in ipairs{0, -1.1, 1.9, 1/0, -1/0, 1e20, -1e20, 0.1, 2000.7} do
    assert(unpack("n", pack("n", n)) == n)
    assert(unpack("<n", pack("<n", n)) == n)
    assert(unpack(">n", pack(">n", n)) == n)
    assert(pack("<f", n) == pack(">f", n):reverse())
    assert(pack(">d", n) == pack("<d", n):reverse())
end

-- for non-native precisions, test only with "round" numbers
for _, n in ipairs{0, -1.5, 1/0, -1/0, 1e10, -1e9, 0.5, 2000.25} do
  assert(unpack("<f", pack("<f", n)) == n)
  assert(unpack(">f", pack(">f", n)) == n)
  assert(unpack("<d", pack("<d", n)) == n)
  assert(unpack(">d", pack(">d", n)) == n)
end
end
