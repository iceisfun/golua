-- Test: Lexer errors for malformed strings
-- From: literals.lua
-- What: Tests that load correctly rejects various forms of malformed strings (non-ending string, invalid escape, invalid long string opener).

do
  assert(not load"a = 'non-ending string")
  assert(not load"a = 'non-ending string\n'")
  assert(not load"a = '\\345'")
  assert(not load"a = [=x]")
end
