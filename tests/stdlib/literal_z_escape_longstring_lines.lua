-- Test: String \z escape and long string line counting
-- From: literals.lua
-- What: Tests the \z escape (which skips following whitespace) and verifies that the parser correctly tracks line numbers in strings and long strings.

do
  local function lexstring (x, y, n)
    local f = assert(load('return ' .. x ..
              ', require"debug".getinfo(1).currentline', ''))
    local s, l = f()
    assert(s == y and l == n)
  end

  lexstring("'abc\\z  \n   efg'", "abcefg", 2)
  lexstring("'abc\\z  \n\n\n'", "abc", 4)
  lexstring("'\\z  \n\t\f\v\n'",  "", 3)
  lexstring("[[\nalo\nalo\n\n]]", "alo\nalo\n\n", 5)
  lexstring("[[\nalo\ralo\n\n]]", "alo\nalo\n\n", 5)
  lexstring("[[\nalo\ralo\r\n]]", "alo\nalo\n", 4)
  lexstring("[[\ralo\n\ralo\r\n]]", "alo\nalo\n", 4)
  lexstring("[[alo]\n]alo]]", "alo]\n]alo", 2)

  assert("abc\z
          def\z
          ghi\z
         " == 'abcdefghi')
end
