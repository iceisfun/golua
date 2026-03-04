-- Test: Platform size detection
-- From: tpack.lua
-- What: Tests that string.packsize returns consistent platform type sizes and verifies basic size ordering constraints.

do
local pack = string.pack
local packsize = string.packsize
local unpack = string.unpack

local NB = 16

local sizeshort = packsize("h")
local sizeint = packsize("i")
local sizelong = packsize("l")
local sizesize_t = packsize("T")
local sizeLI = packsize("j")
local sizefloat = packsize("f")
local sizedouble = packsize("d")
local sizenumber = packsize("n")
local little = (pack("i2", 1) == "\1\0")
local align = packsize("!xXi16")

assert(1 <= sizeshort and sizeshort <= sizeint and sizeint <= sizelong and
       sizefloat <= sizedouble)
end
