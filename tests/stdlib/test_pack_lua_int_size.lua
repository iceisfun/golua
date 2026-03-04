-- Test: Lua integer size (j/J format)
-- From: tpack.lua
-- What: Tests pack/unpack of Lua's native integer type using j (signed) and J (unsigned) format codes at maxinteger/mininteger boundaries.

do
local pack = string.pack
local unpack = string.unpack

assert(unpack(">j", pack(">j", math.maxinteger)) == math.maxinteger)
assert(unpack("<j", pack("<j", math.mininteger)) == math.mininteger)
assert(unpack("<J", pack("<j", -1)) == -1)   -- maximum unsigned integer
end
