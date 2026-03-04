-- Test: Supplementary characters
-- From: utf8.lua
-- What: Tests supplementary Unicode characters (above U+FFFF) including CJK Extension B.

do
local utf8 = require'utf8'

local function len (s)
  return #string.gsub(s, "[\x80-\xBF]", "")
end

local justone = "^" .. utf8.charpattern .. "$"

local function checksyntax (s, t)
  local ts = {"return '"}
  for i = 1, #t do ts[i + 1] = string.format("\\u{%x}", t[i]) end
  ts[#t + 2] = "'"
  ts = table.concat(ts)
  assert(assert(load(ts))() == s)
end

local function check (s, t, nonstrict)
  local l = utf8.len(s, 1, -1, nonstrict)
  assert(#t == l and len(s) == l)
  assert(utf8.char(table.unpack(t)) == s)

  assert(utf8.offset(s, 0) == 1)

  checksyntax(s, t)

  local t1 = {utf8.codepoint(s, 1, -1, nonstrict)}
  assert(#t == #t1)
  for i = 1, #t do assert(t[i] == t1[i]) end

  for i = 1, l do
    local pi = utf8.offset(s, i)
    local pi1 = utf8.offset(s, 2, pi)
    assert(string.find(string.sub(s, pi, pi1 - 1), justone))
    assert(utf8.offset(s, -1, pi1) == pi)
    assert(utf8.offset(s, i - l - 1) == pi)
    assert(pi1 - pi == #utf8.char(utf8.codepoint(s, pi, pi, nonstrict)))
    for j = pi, pi1 - 1 do
      assert(utf8.offset(s, 0, j) == pi)
    end
    for j = pi + 1, pi1 - 1 do
      assert(not utf8.len(s, j))
    end
   assert(utf8.len(s, pi, pi, nonstrict) == 1)
   assert(utf8.len(s, pi, pi1 - 1, nonstrict) == 1)
   assert(utf8.len(s, pi, -1, nonstrict) == l - i + 1)
   assert(utf8.len(s, pi1, -1, nonstrict) == l - i)
   assert(utf8.len(s, 1, pi, nonstrict) == i)
  end

  local i = 0
  for p, c in utf8.codes(s, nonstrict) do
    i = i + 1
    assert(c == t[i] and p == utf8.offset(s, i))
    assert(utf8.codepoint(s, p, p, nonstrict) == c)
  end
  assert(i == #t)

  i = 0
  for c in string.gmatch(s, utf8.charpattern) do
    i = i + 1
    assert(c == utf8.char(t[i]))
  end
  assert(i == #t)

  for i = 1, l do
    assert(utf8.offset(s, i) == utf8.offset(s, i - l - 1, #s + 1))
  end
end

check("𣲷𠜎𠱓𡁻𠵼ab𠺢",
      {0x23CB7, 0x2070E, 0x20C53, 0x2107B, 0x20D7C, 0x61, 0x62, 0x20EA2,})

check("𨳊𩶘𦧺𨳒𥄫𤓓\xF4\x8F\xBF\xBF",
      {0x28CCA, 0x29D98, 0x269FA, 0x28CD2, 0x2512B, 0x244D3, 0x10ffff})
end
