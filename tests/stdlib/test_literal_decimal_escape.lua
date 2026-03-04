-- Test: Decimal escape sequences
-- From: literals.lua
-- What: Tests decimal escape sequences in strings (\DDD form) including boundary behavior and concatenation with embedded nulls.

do
  -- assume ASCII just for tests:
  assert("\09912" == 'c12')
  assert("\99ab" == 'cab')
  assert("\099" == '\99')
  assert("\099\n" == 'c\10')
  assert('\0\0\0alo' == '\0' .. '\0\0' .. 'alo')

  assert(010 .. 020 .. -030 == "1020-30")
end
