-- Test: Large Lua integer pack/unpack
-- From: tpack.lua
-- What: Tests packing/unpacking large Lua integers with extra sign-extension bytes, and overflow detection for integers that do not fit in the target format.
-- Broken: GoLua lexer converts oversized hex literals (>64 bits) to float instead of wrapping to int64

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
  local lnum = 0x13121110090807060504030201
  local s = pack("<j", lnum)
  assert(unpack("<j", s) == lnum)
  assert(unpack("<i" .. sizeLI + 1, s .. "\0") == lnum)
  assert(unpack("<i" .. sizeLI + 1, s .. "\0") == lnum)

  for i = sizeLI + 1, NB do
    local s = pack("<j", -lnum)
    assert(unpack("<j", s) == -lnum)
    -- strings with (correct) extra bytes
    assert(unpack("<i" .. i, s .. ("\xFF"):rep(i - sizeLI)) == -lnum)
    assert(unpack(">i" .. i, ("\xFF"):rep(i - sizeLI) .. s:reverse()) == -lnum)
    assert(unpack("<I" .. i, s .. ("\0"):rep(i - sizeLI)) == -lnum)

    -- overflows
    checkerror("does not fit", unpack, "<I" .. i, ("\x00"):rep(i - 1) .. "\1")
    checkerror("does not fit", unpack, ">i" .. i, "\1" .. ("\x00"):rep(i - 1))
  end
end
end
