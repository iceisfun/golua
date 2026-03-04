-- Test: Fixed-length string packing (c format)
-- From: tpack.lua
-- What: Tests pack/unpack of fixed-length strings (c0, c3, c8, c88, c188), zero-padding behavior, combined with z format, and error for strings longer than the specified length.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

do
  assert(pack("c0", "") == "")
  assert(packsize("c0") == 0)
  assert(unpack("c0", "") == "")
  assert(pack("<! c3", "abc") == "abc")
  assert(packsize("<! c3") == 3)
  assert(pack(">!4 c6", "abcdef") == "abcdef")
  assert(pack("c3", "123") == "123")
  assert(pack("c0", "") == "")
  assert(pack("c8", "123456") == "123456\0\0")
  assert(pack("c88", "") == string.rep("\0", 88))
  assert(pack("c188", "ab") == "ab" .. string.rep("\0", 188 - 2))
  local a, b, c = unpack("!4 z c3", "abcdefghi\0xyz")
  assert(a == "abcdefghi" and b == "xyz" and c == 14)
  checkerror("longer than", pack, "c3", "1234")
end
end
