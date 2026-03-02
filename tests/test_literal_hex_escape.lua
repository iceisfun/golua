-- Test: Hexadecimal escape sequences
-- From: literals.lua
-- What: Tests hexadecimal escape sequences (\xHH form) in strings.

do
  assert("\x00\x05\x10\x1f\x3C\xfF\xe8" == "\0\5\16\31\60\255\232")
end
