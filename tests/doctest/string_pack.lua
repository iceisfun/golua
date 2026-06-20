-- string.pack / string.unpack / string.packsize
-- Binary data packing following Lua 5.4 format strings.

-- Basic integer packing (little-endian by default)
do
    local packed = string.pack("i4", 42)
    print(#packed)
    --> =4
    print(string.unpack("i4", packed))
    --> =42	5
end

-- Unsigned byte round-trip
do
    local a, b = string.unpack("BB", string.pack("BB", 0, 255))
    print(a, b)
    --> =0	255
end

-- Signed/unsigned short
do
    local s = string.pack("<h", -1)
    print(string.unpack("<h", s))
    --> =-1	3
    print(string.unpack("<H", s))
    --> =65535	3
end

-- Big-endian 32-bit integer
do
    local s = string.pack(">i4", 0x01020304)
    local hex = {}
    for i = 1, #s do hex[#hex+1] = string.format("%02x", s:byte(i)) end
    print(table.concat(hex))
    --> =01020304
end

-- Float and double round-trip
do
    local f = string.unpack("f", string.pack("f", 3.14))
    -- float precision: approximately 3.14
    print(string.format("%.2f", f))
    --> =3.14
    print(string.unpack("d", string.pack("d", math.pi)))
    --> =3.1415926535898	9
end

-- Lua number type ('n')
do
    print(string.unpack("n", string.pack("n", 42.5)))
    --> =42.5	9
end

-- Zero-terminated string
do
    local s = string.pack("z", "hello")
    print(#s)
    --> =6
    print(string.unpack("z", s))
    --> =hello	7
end

-- Fixed-size string with zero padding
do
    local s = string.pack("c8", "hi")
    print(#s)
    --> =8
    -- unpack c8 returns all 8 bytes (including null padding)
    local val, pos = string.unpack("c8", s)
    print(#val, pos)
    --> =8	9
end

-- Length-prefixed string
do
    local s = string.pack("s1", "abc")
    print(#s)
    --> =4
    print(string.unpack("s1", s))
    --> =abc	5
end

-- packsize for fixed formats
do
    print(string.packsize("b"))
    --> =1
    print(string.packsize("i4"))
    --> =4
    print(string.packsize("d"))
    --> =8
    print(string.packsize("c10"))
    --> =10
end

-- Mixed endian in one format string
do
    local s = string.pack(">i2 <i2", 10, 20)
    local a, b = string.unpack(">i2 <i2", s)
    print(a, b)
    --> =10	20
end

-- Multiple values
do
    local s = string.pack("bBhHi4", -1, 200, -1000, 60000, 123456)
    local a, b, c, d, e = string.unpack("bBhHi4", s)
    print(a, b, c, d, e)
    --> =-1	200	-1000	60000	123456
end

-- packsize: alignment power-of-2 check on 's' size must fire before the
-- variable-length error (parity with reference Lua's getdetails ordering).
do
    -- 's6' under '!16': prefix size 6 is not a power of 2 -> alignment error wins
    print(pcall(string.packsize, "!16s6"))
    --> =false	bad argument #1 to 'string.packsize' (format asks for alignment not power of 2)
    -- 's4' has power-of-2 prefix -> variable-length error still reported
    print(pcall(string.packsize, "!16s4"))
    --> =false	bad argument #1 to 'string.packsize' (variable-length format)
    -- 'z' needs no alignment -> variable-length error reported
    print(pcall(string.packsize, "!16z"))
    --> =false	bad argument #1 to 'string.packsize' (variable-length format)
    -- s8 (power of 2) -> variable-length
    print(pcall(string.packsize, "!16s8"))
    --> =false	bad argument #1 to 'string.packsize' (variable-length format)
end
