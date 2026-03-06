package tests

import (
	"testing"
)

// diffTest runs the same Lua code in both GoLua and ref Lua, comparing output.
func diffTest(t *testing.T, code string) {
	t.Helper()
	goOut, goErr := runGoLua(t, code)
	refOut, refErr := runRefLua(t, code)
	if refErr != nil {
		t.Skipf("lua5.4 not available: %v", refErr)
	}
	if goErr != nil {
		t.Fatalf("GoLua error: %v\nOutput: %q", goErr, goOut)
	}
	if goOut != refOut {
		t.Errorf("output differs:\n  GoLua: %q\n  Ref:   %q", goOut, refOut)
	}
}

func TestPackBasicIntegers(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
-- Signed/unsigned byte
print(unpack("B", pack("B", 0xff)))
print(unpack("b", pack("b", 0x7f)))
print(unpack("b", pack("b", -0x80)))
-- Short
print(unpack("H", pack("H", 0xffff)))
print(unpack("h", pack("h", 0x7fff)))
print(unpack("h", pack("h", -0x8000)))
-- Long
print(unpack("L", pack("L", 0xffffffff)))
print(unpack("l", pack("l", 0x7fffffff)))
print(unpack("l", pack("l", -0x80000000)))
`)
}

func TestPackVariableWidthIntegers(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local packsize = string.packsize
for i = 1, 16 do
    local s = string.rep("\xff", i)
    assert(pack("i" .. i, -1) == s, "pack i"..i.." -1 failed")
    assert(packsize("i" .. i) == #s)
    assert(unpack("i" .. i, s) == -1)
    s = "\xAA" .. string.rep("\0", i - 1)
    assert(pack("<I" .. i, 0xAA) == s)
    assert(unpack("<I" .. i, s) == 0xAA)
    assert(pack(">I" .. i, 0xAA) == s:reverse())
    assert(unpack(">I" .. i, s:reverse()) == 0xAA)
end
print("OK")
`)
}

func TestPackEndianness(t *testing.T) {
	diffTest(t, `
print(string.pack(">i2 <i2", 10, 20) == "\0\10\20\0")
local a, b = string.unpack("<i2 >i2", "\10\0\0\20")
print(a, b)
print(string.pack("=i4", 2001) == string.pack("i4", 2001))
`)
}

func TestPackFloats(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
for _, n in ipairs{0, -1.1, 1.9, 1/0, -1/0, 1e20, -1e20, 0.1, 2000.7} do
    assert(unpack("n", pack("n", n)) == n)
    assert(unpack("<n", pack("<n", n)) == n)
    assert(unpack(">n", pack(">n", n)) == n)
    assert(pack("<f", n) == pack(">f", n):reverse())
    assert(pack(">d", n) == pack("<d", n):reverse())
end
print("OK")
`)
}

func TestPackStrings(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local s = string.rep("abc", 100)
assert(pack("zB", s, 247) == s .. "\0\xF7")
local s1, b = unpack("zB", s .. "\0\xF9")
assert(b == 249 and s1 == s)
s1 = pack("s", s)
assert(unpack("s", s1) == s)
-- fixed-size strings
assert(pack("c0", "") == "")
assert(pack("c3", "123") == "123")
assert(pack("c8", "123456") == "123456\0\0")
assert(unpack("c3", "abcdef") == "abc")
print("OK")
`)
}

func TestPackAlignment(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local packsize = string.packsize
assert(pack(" < i1 i2 ", 2, 3) == "\2\3\0")
local x = pack(">!8 b Xh i4 i8 c1 Xi8", -12, 100, 200, "\xEC")
assert(#x == packsize(">!8 b Xh i4 i8 c1 Xi8"))
assert(x == "\xf4" .. "\0\0\0" ..
            "\0\0\0\100" ..
            "\0\0\0\0\0\0\0\xC8" ..
            "\xEC" .. "\0\0\0\0\0\0\0")
print("OK")
`)
}

func TestPackOverflow(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local sizeLI = string.packsize("j")
local function checkerror(msg, f, ...)
    local ok, err = pcall(f, ...)
    assert(not ok and string.find(err, msg), "expected " .. msg .. " got " .. tostring(err))
end
for i = 1, sizeLI - 1 do
    local umax = (1 << (i * 8)) - 1
    local max = umax >> 1
    local min = ~max
    checkerror("overflow", pack, "<I" .. i, -1)
    checkerror("overflow", pack, "<I" .. i, min)
    checkerror("overflow", pack, ">I" .. i, umax + 1)
    checkerror("overflow", pack, ">i" .. i, umax)
    checkerror("overflow", pack, ">i" .. i, max + 1)
    checkerror("overflow", pack, "<i" .. i, min - 1)
end
print("OK")
`)
}

func TestPackInvalidFormats(t *testing.T) {
	diffTest(t, `
local function checkerror(msg, f, ...)
    local ok, err = pcall(f, ...)
    assert(not ok and string.find(err, msg), "expected " .. msg .. " got " .. tostring(err))
end
checkerror("out of limits", string.pack, "i0", 0)
checkerror("out of limits", string.pack, "i17", 0)
checkerror("out of limits", string.pack, "!17", 0)
checkerror("invalid format option 'r'", string.pack, "i3r", 0)
checkerror("missing size", string.pack, "c", "")
checkerror("variable%-length format", string.packsize, "s")
checkerror("variable%-length format", string.packsize, "z")
print("OK")
`)
}

func TestPackInitialPosition(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local x = pack("i4i4i4i4", 1, 2, 3, 4)
for pos = 1, 16, 4 do
    local i, p = unpack("i4", x, pos)
    assert(i == pos//4 + 1 and p == pos + 4)
end
-- negative indices
local i, p = unpack("!4 i4", x, -4)
assert(i == 4 and p == 17)
local i2, p2 = unpack("!4 i4", x, -#x)
assert(i2 == 1 and p2 == 5)
print("OK")
`)
}

func TestPackMultiTypes(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local packsize = string.packsize
local x = pack("<b h b f d f n i", 1, 2, 3, 4, 5, 6, 7, 8)
assert(#x == packsize("<b h b f d f n i"))
local a, b, c, d, e, f, g, h = unpack("<b h b f d f n i", x)
assert(a == 1 and b == 2 and c == 3 and d == 4 and e == 5 and f == 6 and
       g == 7 and h == 8)
print("OK")
`)
}

func TestPackPlatformSizes(t *testing.T) {
	diffTest(t, `
local packsize = string.packsize
print(packsize("h"))
print(packsize("i"))
print(packsize("l"))
print(packsize("T"))
print(packsize("j"))
print(packsize("f"))
print(packsize("d"))
print(packsize("n"))
`)
}

func TestPackLuaIntegerBoundary(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
assert(unpack(">j", pack(">j", math.maxinteger)) == math.maxinteger)
assert(unpack("<j", pack("<j", math.mininteger)) == math.mininteger)
assert(unpack("<J", pack("<j", -1)) == -1)
print("OK")
`)
}

func TestPackSignExtension(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local sizeLI = string.packsize("j")
local u = 0xf0
for i = 1, sizeLI - 1 do
    assert(unpack("<i"..i, "\xf0"..("\xff"):rep(i - 1)) == -16)
    assert(unpack(">I"..i, "\xf0"..("\xff"):rep(i - 1)) == u)
    u = u * 256 + 0xff
end
print("OK")
`)
}

func TestPackXAlignment(t *testing.T) {
	diffTest(t, `
local packsize = string.packsize
local unpack = string.unpack
assert(packsize("!8 xXi8") == 8)
local pos = unpack("!8 xXi8", "0123456701234567"); assert(pos == 9)
assert(packsize("!8 xXi2") == 2)
pos = unpack("!8 xXi2", "0123456701234567"); assert(pos == 3)
assert(packsize("!2 xXi2") == 2)
pos = unpack("!2 xXi2", "0123456701234567"); assert(pos == 3)
assert(packsize("!16 xXi16") == 16)
pos = unpack("!16 xXi16", "0123456701234567"); assert(pos == 17)
print("OK")
`)
}

func TestPackSizedStrings(t *testing.T) {
	diffTest(t, `
local pack = string.pack
local unpack = string.unpack
local s = string.rep("abc", 100)
for i = 2, 16 do
    local s1 = pack("s" .. i, s)
    assert(unpack("s" .. i, s1) == s and #s1 == #s + i)
end
print("OK")
`)
}

func TestPackErrorMessages(t *testing.T) {
	diffTest(t, `
local function check(msg, f, ...)
    local ok, err = pcall(f, ...)
    assert(not ok, "expected error")
    assert(string.find(err, msg, 1, true), "expected '"..msg.."' in: "..tostring(err))
end
check("contains zeros", string.pack, "z", "alo\0")
check("does not fit", string.pack, "s1", string.rep("x", 300))
check("longer than", string.pack, "c3", "1234")
check("too short", string.unpack, "c5", "abcd")
check("invalid next option", string.pack, "X")
check("invalid next option", string.pack, "Xc1")
print("OK")
`)
}
