-- string.pack/unpack/packsize: binary data packing (Lua 5.3+)
assert(string.pack ~= nil, "string.pack should exist")
assert(string.unpack ~= nil, "string.unpack should exist")
assert(string.packsize ~= nil, "string.packsize should exist")

-- Basic pack/unpack round-trip
local packed = string.pack("bB", 100, 200)
local a, b = string.unpack("bB", packed)
assert(a == 100 and b == 200, "pack/unpack round-trip")

-- packsize
assert(string.packsize("bb") == 2, "packsize bb")
