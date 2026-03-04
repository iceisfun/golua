-- Test: Escape sequences in strings
-- From: literals.lua
-- What: Tests basic escape sequences: newline, double quote, single quote, backslash, and control characters.

do
  assert('\n\"\'\\' == [[

"'\]])

  assert(string.find("\a\b\f\n\r\t\v", "^%c%c%c%c%c%c%c$"))
end
