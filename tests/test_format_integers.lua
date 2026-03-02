-- Test: string.format integer specifiers (%x, %d, %u, %o)
-- From: strings.lua
-- What: Tests string.format with hex, decimal, unsigned, and octal integer formatting, including 32-bit and 64-bit edge cases with sign/padding.

do
assert(string.format("%x", 0.0) == "0")
assert(string.format("%02x", 0.0) == "00")
assert(string.format("%08X", 0xFFFFFFFF) == "FFFFFFFF")
assert(string.format("%+08d", 31501) == "+0031501")
assert(string.format("%+08d", -30927) == "-0030927")

do    -- longest number that can be formatted
  local i = 1
  local j = 10000
  while i + 1 < j do
    local m = (i + j) // 2
    if 10^m < math.huge then i = m else j = m end
  end
  assert(10^i < math.huge and 10^j == math.huge)
  local s = string.format('%.99f', -(10^i))
  assert(string.len(s) >= i + 101)
  assert(tonumber(s) == -(10^i))

  assert(10^38 < math.huge)
  local s = string.format('%.99f', -(10^38))
  assert(string.len(s) >= 38 + 101)
  assert(tonumber(s) == -(10^38))
end

do   -- assume at least 32 bits
  local max, min = 0x7fffffff, -0x80000000
  assert(string.sub(string.format("%8x", -1), -8) == "ffffffff")
  assert(string.format("%x", max) == "7fffffff")
  assert(string.sub(string.format("%x", min), -8) == "80000000")
  assert(string.format("%d", max) ==  "2147483647")
  assert(string.format("%d", min) == "-2147483648")
  assert(string.format("%u", 0xffffffff) == "4294967295")
  assert(string.format("%o", 0xABCD) == "125715")

  max, min = 0x7fffffffffffffff, -0x8000000000000000
  if max > 2.0^53 then  -- only for 64 bits
    assert(string.format("%x", (2^52 | 0) - 1) == "fffffffffffff")
    assert(string.format("0x%8X", 0x8f000003) == "0x8F000003")
    assert(string.format("%d", 2^53) == "9007199254740992")
    assert(string.format("%i", -2^53) == "-9007199254740992")
    assert(string.format("%x", max) == "7fffffffffffffff")
    assert(string.format("%x", min) == "8000000000000000")
    assert(string.format("%d", max) ==  "9223372036854775807")
    assert(string.format("%d", min) == "-9223372036854775808")
    assert(string.format("%u", ~(-1 << 64)) == "18446744073709551615")
    assert(tostring(1234567890123) == '1234567890123')
  end
end
end
