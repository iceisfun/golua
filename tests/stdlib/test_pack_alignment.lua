-- Test: Alignment
-- From: tpack.lua
-- What: Tests alignment directives (!, !4, !8, !16) with various combinations of integer, byte, string, and padding (x, X) formats, verifying correct padding byte insertion and total size.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

do
  assert(pack(" < i1 i2 ", 2, 3) == "\2\3\0")   -- no alignment by default
  local x = pack(">!8 b Xh i4 i8 c1 Xi8", -12, 100, 200, "\xEC")
  assert(#x == packsize(">!8 b Xh i4 i8 c1 Xi8"))
  assert(x == "\xf4" .. "\0\0\0" ..
              "\0\0\0\100" ..
              "\0\0\0\0\0\0\0\xC8" ..
              "\xEC" .. "\0\0\0\0\0\0\0")
  local a, b, c, d, pos = unpack(">!8 c1 Xh i4 i8 b Xi8 XI XH", x)
  assert(a == "\xF4" and b == 100 and c == 200 and d == -20 and (pos - 1) == #x)

  x = pack(">!4 c3 c4 c2 z i4 c5 c2 Xi4",
                  "abc", "abcd", "xz", "hello", 5, "world", "xy")
  assert(x == "abcabcdxzhello\0\0\0\0\0\5worldxy\0")
  local a, b, c, d, e, f, g, pos = unpack(">!4 c3 c4 c2 z i4 c5 c2 Xh Xi4", x)
  assert(a == "abc" and b == "abcd" and c == "xz" and d == "hello" and
         e == 5 and f == "world" and g == "xy" and (pos - 1) % 4 == 0)

  x = pack(" b b Xd b Xb x", 1, 2, 3)
  assert(packsize(" b b Xd b Xb x") == 4)
  assert(x == "\1\2\3\0")
  a, b, c, pos = unpack("bbXdb", x)
  assert(a == 1 and b == 2 and c == 3 and pos == #x)

  -- only alignment
  assert(packsize("!8 xXi8") == 8)
  local pos = unpack("!8 xXi8", "0123456701234567"); assert(pos == 9)
  assert(packsize("!8 xXi2") == 2)
  local pos = unpack("!8 xXi2", "0123456701234567"); assert(pos == 3)
  assert(packsize("!2 xXi2") == 2)
  local pos = unpack("!2 xXi2", "0123456701234567"); assert(pos == 3)
  assert(packsize("!2 xXi8") == 2)
  local pos = unpack("!2 xXi8", "0123456701234567"); assert(pos == 3)
  assert(packsize("!16 xXi16") == 16)
  local pos = unpack("!16 xXi16", "0123456701234567"); assert(pos == 17)

  checkerror("invalid next option", pack, "X")
  checkerror("invalid next option", unpack, "XXi", "")
  checkerror("invalid next option", unpack, "X i", "")
  checkerror("invalid next option", pack, "Xc1")
end
end
