-- test_utf8_lax.lua
-- utf8.len/codepoint should accept the lax flag and behave identically on valid data.

local sample = "héllo"

local len_strict = utf8.len(sample)
local len_lax = utf8.len(sample, 1, -1, true)
assert(len_strict == len_lax, string.format("expected lax len to equal strict len (%d vs %d)", len_strict, len_lax))

local c1, c2 = utf8.codepoint(sample, 1, 2)
local l1, l2 = utf8.codepoint(sample, 1, 2, true)
assert(c1 == l1 and c2 == l2, "utf8.codepoint should accept lax flag on valid data")

for _, s in ipairs({
  string.char(0xC0, 0x80),
  string.char(0xE0, 0x80, 0x80),
  string.char(0xF0, 0x80, 0x80, 0x80),
  string.char(0xF8, 0x80, 0x80, 0x80, 0x80),
  string.char(0xFC, 0x80, 0x80, 0x80, 0x80, 0x80),
}) do
  local n, pos = utf8.len(s, 1, -1, true)
  assert(n == nil and pos == 1, "utf8.len lax should reject overlong encodings")
  local ok = pcall(utf8.codepoint, s, 1, #s, true)
  assert(not ok, "utf8.codepoint lax should reject overlong encodings")
end
