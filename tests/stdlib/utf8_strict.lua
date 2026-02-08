-- utf8_strict.lua: Tests for the utf8 library (strict mode only).

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- utf8.char: encode codepoints to UTF-8
--------------------------------------------------------------------------------

-- ASCII
assert_eq(utf8.char(72, 101, 108, 108, 111), "Hello", "utf8.char ASCII")
assert_eq(utf8.char(0), "\0", "utf8.char NUL")
assert_eq(utf8.char(127), "\x7F", "utf8.char DEL")

-- Multibyte: verify via round-trip
local latin = utf8.char(0xE4, 0xE5)
assert_eq(utf8.len(latin), 2, "utf8.char latin len")
local lc1, lc2 = utf8.codepoint(latin, 1, #latin)
assert_eq(lc1, 0xE4, "utf8.char latin cp1")
assert_eq(lc2, 0xE5, "utf8.char latin cp2")

assert_eq(utf8.char(0x4F60, 0x597D), "你好", "utf8.char Chinese")

local emoji = utf8.char(0x1F600)
assert_eq(utf8.len(emoji), 1, "utf8.char emoji len")
assert_eq(utf8.codepoint(emoji), 0x1F600, "utf8.char emoji codepoint")

-- Boundary: U+10FFFF (max valid Unicode)
assert(utf8.char(0x10FFFF) ~= nil, "utf8.char U+10FFFF should succeed")

-- Zero arguments
assert_eq(utf8.char(), "", "utf8.char no args")

-- Error: value out of range (> U+10FFFF)
local ok, err = pcall(utf8.char, 0x110000)
assert(not ok, "utf8.char > U+10FFFF must error")

-- Error: surrogate half
local ok2, err2 = pcall(utf8.char, 0xD800)
assert(not ok2, "utf8.char surrogate must error")

local ok3, err3 = pcall(utf8.char, 0xDFFF)
assert(not ok3, "utf8.char surrogate end must error")

-- Error: negative
local ok4, err4 = pcall(utf8.char, -1)
assert(not ok4, "utf8.char negative must error")

--------------------------------------------------------------------------------
-- utf8.len: count UTF-8 characters
--------------------------------------------------------------------------------

-- ASCII
assert_eq(utf8.len("Hello"), 5, "utf8.len ASCII")
assert_eq(utf8.len(""), 0, "utf8.len empty")
assert_eq(utf8.len("a"), 1, "utf8.len single ASCII")

-- Multibyte
assert_eq(utf8.len("你好"), 2, "utf8.len Chinese")
assert_eq(utf8.len("café"), 4, "utf8.len mixed latin")
assert_eq(utf8.len("日本語"), 3, "utf8.len Japanese")

-- Mixed ASCII and multibyte
assert_eq(utf8.len("Hello, 世界!"), 10, "utf8.len mixed")

-- Emoji (4-byte sequences)
local emoji_s = utf8.char(0x1F600)
assert_eq(utf8.len(emoji_s), 1, "utf8.len emoji")

-- Subrange with i and j
assert_eq(utf8.len("Hello", 1, 3), 3, "utf8.len subrange")
assert_eq(utf8.len("Hello", 2, 4), 3, "utf8.len subrange mid")

-- Negative indices
assert_eq(utf8.len("Hello", 1, -1), 5, "utf8.len negative j")
assert_eq(utf8.len("Hello", -3, -1), 3, "utf8.len negative i and j")

-- i = #s + 1 yields 0
assert_eq(utf8.len("abc", 4), 0, "utf8.len i past end")

-- Invalid UTF-8: soft fail returns nil + position
-- Note: use string.char() to create raw bytes (golua's \xNN encodes as UTF-8)
local l, p = utf8.len(string.char(0xFF))
assert(l == nil, "utf8.len invalid returns nil")
assert_eq(p, 1, "utf8.len invalid position")

local l2, p2 = utf8.len("abc" .. string.char(0xFF) .. "def")
assert(l2 == nil, "utf8.len invalid mid returns nil")
assert_eq(p2, 4, "utf8.len invalid mid position")

-- Invalid: continuation byte alone
local l3, p3 = utf8.len(string.char(0x80))
assert(l3 == nil, "utf8.len bare continuation returns nil")
assert_eq(p3, 1, "utf8.len bare continuation position")

-- Invalid: truncated multibyte
local l4, p4 = utf8.len(string.char(0xC3))
assert(l4 == nil, "utf8.len truncated 2-byte returns nil")
assert_eq(p4, 1, "utf8.len truncated 2-byte position")

--------------------------------------------------------------------------------
-- utf8.codepoint: extract codepoints as integers
--------------------------------------------------------------------------------

-- ASCII
assert_eq(utf8.codepoint("A"), 65, "utf8.codepoint A")
assert_eq(utf8.codepoint("Hello", 1, 1), 72, "utf8.codepoint H")

-- Multiple returns
local a, b, c = utf8.codepoint("ABC", 1, 3)
assert_eq(a, 65, "utf8.codepoint A multi")
assert_eq(b, 66, "utf8.codepoint B multi")
assert_eq(c, 67, "utf8.codepoint C multi")

-- Multibyte
assert_eq(utf8.codepoint("你"), 0x4F60, "utf8.codepoint Chinese")

-- Default j = i (single char)
assert_eq(utf8.codepoint("ABC", 2), 66, "utf8.codepoint default j=i")

-- Negative index
assert_eq(utf8.codepoint("ABC", -1), 67, "utf8.codepoint negative i")

-- Empty interval returns nothing
local results = {utf8.codepoint("ABC", 3, 2)}
assert_eq(#results, 0, "utf8.codepoint empty interval")

-- Error on invalid UTF-8
local ok5, err5 = pcall(utf8.codepoint, string.char(0xFF), 1, 1)
assert(not ok5, "utf8.codepoint invalid must error")

--------------------------------------------------------------------------------
-- utf8.codes: iterator
--------------------------------------------------------------------------------

-- ASCII iteration
local positions = {}
local codepoints = {}
for p, c in utf8.codes("ABC") do
    positions[#positions + 1] = p
    codepoints[#codepoints + 1] = c
end
assert_eq(#positions, 3, "utf8.codes ASCII count")
assert_eq(positions[1], 1, "utf8.codes pos 1")
assert_eq(positions[2], 2, "utf8.codes pos 2")
assert_eq(positions[3], 3, "utf8.codes pos 3")
assert_eq(codepoints[1], 65, "utf8.codes cp A")
assert_eq(codepoints[2], 66, "utf8.codes cp B")
assert_eq(codepoints[3], 67, "utf8.codes cp C")

-- Multibyte iteration
local mpos = {}
local mcp = {}
for p, c in utf8.codes("你好") do
    mpos[#mpos + 1] = p
    mcp[#mcp + 1] = c
end
assert_eq(#mpos, 2, "utf8.codes multibyte count")
assert_eq(mpos[1], 1, "utf8.codes Chinese pos 1")
assert_eq(mpos[2], 4, "utf8.codes Chinese pos 2")
assert_eq(mcp[1], 0x4F60, "utf8.codes Chinese cp 1")
assert_eq(mcp[2], 0x597D, "utf8.codes Chinese cp 2")

-- Empty string
local ecount = 0
for p, c in utf8.codes("") do
    ecount = ecount + 1
end
assert_eq(ecount, 0, "utf8.codes empty string")

-- Error on invalid UTF-8
local ok6, err6 = pcall(function()
    for p, c in utf8.codes("abc" .. string.char(0xFF) .. "def") do end
end)
assert(not ok6, "utf8.codes invalid must error")

--------------------------------------------------------------------------------
-- utf8.offset: byte position from codepoint offset
--------------------------------------------------------------------------------

-- ASCII: byte position = codepoint position
assert_eq(utf8.offset("Hello", 1), 1, "utf8.offset n=1")
assert_eq(utf8.offset("Hello", 2), 2, "utf8.offset n=2")
assert_eq(utf8.offset("Hello", 5), 5, "utf8.offset n=5")
assert_eq(utf8.offset("Hello", 6), 6, "utf8.offset n=6 (past end)")

-- Past end returns nil
assert(utf8.offset("Hello", 7) == nil, "utf8.offset n=7 returns nil")

-- Negative n: count from end
assert_eq(utf8.offset("Hello", -1), 5, "utf8.offset n=-1")
assert_eq(utf8.offset("Hello", -5), 1, "utf8.offset n=-5")
assert(utf8.offset("Hello", -6) == nil, "utf8.offset n=-6 returns nil")

-- n=0: find start of character containing byte i
assert_eq(utf8.offset("Hello", 0, 3), 3, "utf8.offset n=0 ASCII")

-- Multibyte: 你好 is 6 bytes (3+3)
assert_eq(utf8.offset("你好", 1), 1, "utf8.offset Chinese n=1")
assert_eq(utf8.offset("你好", 2), 4, "utf8.offset Chinese n=2")
assert_eq(utf8.offset("你好", 3), 7, "utf8.offset Chinese n=3 (past end)")

-- n=0 with continuation byte: walk back to character start
assert_eq(utf8.offset("你好", 0, 2), 1, "utf8.offset n=0 continuation byte 2")
assert_eq(utf8.offset("你好", 0, 3), 1, "utf8.offset n=0 continuation byte 3")
assert_eq(utf8.offset("你好", 0, 5), 4, "utf8.offset n=0 second char cont")

-- Negative n with multibyte
assert_eq(utf8.offset("你好", -1), 4, "utf8.offset Chinese n=-1")
assert_eq(utf8.offset("你好", -2), 1, "utf8.offset Chinese n=-2")

-- With explicit i parameter
assert_eq(utf8.offset("Hello", 2, 3), 4, "utf8.offset n=2 from i=3")
assert_eq(utf8.offset("Hello", -1, 3), 2, "utf8.offset n=-1 from i=3")

-- Error: continuation byte for n != 0
local ok7, err7 = pcall(utf8.offset, "你好", 1, 2)
assert(not ok7, "utf8.offset continuation byte for n>0 must error")

local ok8, err8 = pcall(utf8.offset, "你好", -1, 2)
assert(not ok8, "utf8.offset continuation byte for n<0 must error")

--------------------------------------------------------------------------------
-- utf8.charpattern
--------------------------------------------------------------------------------

assert_eq(type(utf8.charpattern), "string", "utf8.charpattern is string")
assert_eq(#utf8.charpattern, 14, "utf8.charpattern length")

--------------------------------------------------------------------------------
-- Lax mode acceptance (lax flag is accepted for Lua 5.4 API compatibility)
--------------------------------------------------------------------------------

-- utf8.len with lax=true must succeed on valid UTF-8
local ok9, res9 = pcall(utf8.len, "hello", 1, -1, true)
assert(ok9, "utf8.len lax=true should succeed on valid UTF-8")
assert(res9 == 5, "utf8.len lax=true should return correct length")

-- utf8.codepoint with lax=true must succeed on valid UTF-8
local ok10, res10 = pcall(utf8.codepoint, "hello", 1, 1, true)
assert(ok10, "utf8.codepoint lax=true should succeed on valid UTF-8")
assert(res10 == 104, "utf8.codepoint lax=true should return correct codepoint")

-- utf8.codes with lax=true must succeed on valid UTF-8
local ok11, res11 = pcall(utf8.codes, "hello", true)
assert(ok11, "utf8.codes lax=true should succeed on valid UTF-8")

--------------------------------------------------------------------------------
-- Boundary values
--------------------------------------------------------------------------------

-- U+0080 (2-byte boundary)
local u80 = utf8.char(0x80)
assert_eq(#u80, 2, "utf8.char U+0080 is 2 bytes")
assert_eq(utf8.codepoint(u80), 0x80, "utf8.codepoint U+0080")

-- U+07FF (max 2-byte)
assert_eq(utf8.codepoint(utf8.char(0x7FF)), 0x7FF, "roundtrip U+07FF")

-- U+0800 (3-byte boundary)
assert_eq(utf8.codepoint(utf8.char(0x800)), 0x800, "roundtrip U+0800")

-- U+FFFF (max 3-byte, excluding surrogates)
assert_eq(utf8.codepoint(utf8.char(0xFFFF)), 0xFFFF, "roundtrip U+FFFF")

-- U+10000 (4-byte boundary)
assert_eq(utf8.codepoint(utf8.char(0x10000)), 0x10000, "roundtrip U+10000")

-- U+10FFFF (max valid codepoint)
assert_eq(utf8.codepoint(utf8.char(0x10FFFF)), 0x10FFFF, "roundtrip U+10FFFF")

-- Round-trip: encode then decode
local s = utf8.char(72, 0x4F60, 0x1F600, 65)
local c1, c2, c3, c4 = utf8.codepoint(s, 1, #s)
assert_eq(c1, 72, "roundtrip c1")
assert_eq(c2, 0x4F60, "roundtrip c2")
assert_eq(c3, 0x1F600, "roundtrip c3")
assert_eq(c4, 65, "roundtrip c4")
