-- Test: pm.lua - Null byte in patterns
-- From: pm.lua
-- What: Tests patterns containing `\0` (null bytes) in character sets, balanced matches, and quantifiers.

do
  assert(string.match("ab\0\1\2c", "[\0-\2]+") == "\0\1\2")
  assert(string.match("ab\0\1\2c", "[\0-\0]+") == "\0")
  assert(string.find("b$a", "$\0?") == 2)
  assert(string.find("abc\0efg", "%\0") == 4)
  assert(string.match("abc\0efg\0\1e\1g", "%b\0\1") == "\0efg\0\1e\1")
  assert(string.match("abc\0\0\0", "%\0+") == "\0\0\0")
  assert(string.match("abc\0\0\0", "%\0%\0?") == "\0\0")

  -- magic char after \0
  assert(string.find("abc\0\0","\0.") == 4)
  assert(string.find("abcx\0\0abc\0abc","x\0\0abc\0a.") == 4)
end
