-- Test: math.lua - tonumber invalid formats
-- From: math.lua
-- What: Tests that tonumber returns nil/false for various invalid string formats including embedded NUL bytes, trailing characters, invalid base digits, inf, and nan strings.

do
  local function f (...)
    if select('#', ...) == 1 then
      return (...)
    else
      return "***"
    end
  end

  assert(not f(tonumber('fFfa', 15)))
  assert(not f(tonumber('099', 8)))
  assert(not f(tonumber('1\0', 2)))
  assert(not f(tonumber('', 8)))
  assert(not f(tonumber('  ', 9)))
  assert(not f(tonumber('  ', 9)))
  assert(not f(tonumber('0xf', 10)))

  assert(not f(tonumber('inf')))
  assert(not f(tonumber(' INF ')))
  assert(not f(tonumber('Nan')))
  assert(not f(tonumber('nan')))

  assert(not f(tonumber('  ')))
  assert(not f(tonumber('')))
  assert(not f(tonumber('1  a')))
  assert(not f(tonumber('1  a', 2)))
  assert(not f(tonumber('1\0')))
  assert(not f(tonumber('1 \0')))
  assert(not f(tonumber('1\0 ')))
  assert(not f(tonumber('e1')))
  assert(not f(tonumber('e  1')))
  assert(not f(tonumber(' 3.4.5 ')))
end
