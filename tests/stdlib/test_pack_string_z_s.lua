-- Test: String packing (z and s formats)
-- From: tpack.lua
-- What: Tests pack/unpack of zero-terminated strings (z), length-prefixed strings (s, s1..s16), and related error conditions (contains zeros, unfinished string, does not fit).

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack
local NB = 16
local sizeLI = packsize("j")

local function checkerror (msg, f, ...)
  local status, err = pcall(f, ...)
  assert(not status and string.find(err, msg))
end

do
  local s = string.rep("abc", 1000)
  assert(pack("zB", s, 247) == s .. "\0\xF7")
  local s1, b = unpack("zB", s .. "\0\xF9")
  assert(b == 249 and s1 == s)
  s1 = pack("s", s)
  assert(unpack("s", s1) == s)

  checkerror("does not fit", pack, "s1", s)

  checkerror("contains zeros", pack, "z", "alo\0");

  checkerror("unfinished string", unpack, "zc10000000", "alo")

  for i = 2, NB do
    local s1 = pack("s" .. i, s)
    assert(unpack("s" .. i, s1) == s and #s1 == #s + i)
  end
end

do
  local x = pack("s", "alo")
  checkerror("too short", unpack, "s", x:sub(1, -2))
  checkerror("too short", unpack, "c5", "abcd")
  checkerror("out of limits", pack, "s100", "alo")
end
end
