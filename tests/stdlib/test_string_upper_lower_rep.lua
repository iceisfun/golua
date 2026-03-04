-- Test: string.upper, string.lower, string.rep
-- From: strings.lua
-- What: Tests string.upper/lower with embedded null bytes, string.rep with zero/positive counts and separators, and overflow errors.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

local maxi = math.maxinteger

assert(string.upper("ab\0c") == "AB\0C")
assert(string.lower("\0ABCc%$") == "\0abcc%$")
assert(string.rep('teste', 0) == '')
assert(string.rep('t\xE3s\00t\xE3', 2) == 't\xE3s\0t\xE3t\xE3s\000t\xE3')
assert(string.rep('', 10) == '')

if string.packsize("i") == 4 then
  -- result length would be 2^31 (int overflow)
  checkerror("too large", string.rep, 'aa', (1 << 30))
  checkerror("too large", string.rep, 'a', (1 << 30), ',')
end

-- repetitions with separator
assert(string.rep('teste', 0, 'xuxu') == '')
assert(string.rep('teste', 1, 'xuxu') == 'teste')
assert(string.rep('\1\0\1', 2, '\0\0') == '\1\0\1\0\0\1\0\1')
assert(string.rep('', 10, '.') == string.rep('.', 9))
assert(not pcall(string.rep, "aa", maxi // 2 + 10))
assert(not pcall(string.rep, "", maxi // 2 + 10, "aa"))
end
