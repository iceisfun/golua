-- Test: math.lua - tonumber with very long numerals
-- From: math.lua
-- What: Tests tonumber parsing of hex floats with very long digit strings (up to 500 hex digits) and hex exponent notation.

do
  if not _soft then
    -- tests with very long numerals
    assert(tonumber("0x"..string.rep("f", 13)..".0") == 2.0^(4*13) - 1)
    assert(tonumber("0x"..string.rep("f", 150)..".0") == 2.0^(4*150) - 1)
    assert(tonumber("0x"..string.rep("f", 300)..".0") == 2.0^(4*300) - 1)
    assert(tonumber("0x"..string.rep("f", 500)..".0") == 2.0^(4*500) - 1)
    assert(tonumber('0x3.' .. string.rep('0', 1000)) == 3)
    assert(tonumber('0x' .. string.rep('0', 1000) .. 'a') == 10)
    assert(tonumber('0x0.' .. string.rep('0', 13).."1") == 2.0^(-4*14))
    assert(tonumber('0x0.' .. string.rep('0', 150).."1") == 2.0^(-4*151))
    assert(tonumber('0x0.' .. string.rep('0', 300).."1") == 2.0^(-4*301))
    assert(tonumber('0x0.' .. string.rep('0', 500).."1") == 2.0^(-4*501))

    assert(tonumber('0xe03' .. string.rep('0', 1000) .. 'p-4000') == 3587.0)
    assert(tonumber('0x.' .. string.rep('0', 1000) .. '74p4004') == 0x7.4)
  end
end
