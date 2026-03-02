-- Test: Valid characters in variable names
-- From: literals.lua
-- What: Tests that only valid identifier characters (letters, underscore, digits after first character) are accepted in variable names, checking all 256 byte values.

do
  for i = 0, 255 do
    local s = string.char(i)
    assert(not string.find(s, "[a-zA-Z_]") == not load(s .. "=1", ""))
    assert(not string.find(s, "[a-zA-Z_0-9]") ==
           not load("a" .. s .. "1 = 1", ""))
  end
end
