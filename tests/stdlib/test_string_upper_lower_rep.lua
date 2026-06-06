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

do
  -- Lua 5.5 uses size_t (not 32-bit int) for the result length, so a 2^31-byte
  -- request no longer overflows: reference Lua allocates it when memory allows.
  -- golua's sandbox caps allocation and rejects with "not enough memory" (the
  -- same message reference gives for a representable-but-unallocatable size);
  -- "resulting string too large" is reserved for a genuine size overflow.
  checkerror("not enough memory", string.rep, 'aa', (1 << 30))
  checkerror("not enough memory", string.rep, 'a', (1 << 30), ',')
end

-- repetitions with separator
assert(string.rep('teste', 0, 'xuxu') == '')
assert(string.rep('teste', 1, 'xuxu') == 'teste')
assert(string.rep('\1\0\1', 2, '\0\0') == '\1\0\1\0\0\1\0\1')
assert(string.rep('', 10, '.') == string.rep('.', 9))
assert(not pcall(string.rep, "aa", maxi // 2 + 10))
assert(not pcall(string.rep, "", maxi // 2 + 10, "aa"))
end
