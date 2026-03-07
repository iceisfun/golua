-- Test: tonumber with explicit base wraps on overflow (modulo 2^64)
-- Bug: GoLua returned nil for overflowing values instead of wrapping.
-- Lua 5.4 uses unsigned wrapping arithmetic for tonumber(s, base).

do
  -- Hex overflow: 2^64 wraps to 0
  assert(tonumber("10000000000000000", 16) == 0,
    "hex 2^64 should wrap to 0")

  -- Hex overflow: 2^64 + 1 wraps to 1
  assert(tonumber("10000000000000001", 16) == 1,
    "hex 2^64+1 should wrap to 1")

  -- Max uint64 is fine (not overflow)
  assert(tonumber("ffffffffffffffff", 16) == -1,
    "hex max uint64 should be -1 as signed")

  -- 2 * max uint64 wraps
  assert(tonumber("1ffffffffffffffff", 16) == -1,
    "hex 2*maxuint64 should wrap to -1")

  -- Decimal overflow: 2^64 = 18446744073709551616
  assert(tonumber("18446744073709551616", 10) == 0,
    "decimal 2^64 should wrap to 0")
  assert(tonumber("18446744073709551617", 10) == 1,
    "decimal 2^64+1 should wrap to 1")

  -- Binary overflow: 65 ones = 2^65 - 1
  local s65 = string.rep("1", 65)
  assert(tonumber(s65, 2) == -1,
    "binary 65 ones should wrap to -1")

  -- Negative with base
  assert(tonumber("-1", 16) == -1)
  assert(tonumber("-8000000000000001", 16) == math.maxinteger,
    "negative hex overflow should wrap")
  assert(tonumber("-FFFFFFFFFFFFFFFF", 16) == 1,
    "negative max uint64 should wrap to 1")

  -- Base 36 overflow
  local r1 = tonumber("zzzzzzzzzzzz", 36)
  local r2 = tonumber("zzzzzzzzzzzzz", 36)
  assert(r1 ~= nil, "base 36 overflow should wrap, not nil")
  assert(r2 ~= nil, "base 36 larger overflow should wrap, not nil")

  -- Non-overflow cases should still work
  assert(tonumber("ff", 16) == 255)
  assert(tonumber("111", 2) == 7)
  assert(tonumber("zz", 36) == 1295)

  -- Invalid digits should still return nil
  assert(tonumber("gg", 16) == nil)
  assert(tonumber("", 16) == nil)


end
