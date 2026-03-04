-- Test: pm.lua - Character class patterns
-- From: pm.lua
-- What: Tests character class patterns like `%l`, `%a`, `%g` and various quantifiers (`*`, `+`, `-`, `?`), anchors (`^`, `$`), and pattern escapes.

do
  local function f (s, p)
    local i,e = string.find(s, p)
    if i then return string.sub(s, i, e) end
  end

  assert(f('aloALO', '%l*') == 'alo')
  assert(f('aLo_ALO', '%a*') == 'aLo')

  assert(f("  \n\r*&\n\r   xuxu  \n\n", "%g%g%g+") == "xuxu")

  assert(f('aaab', 'a*') == 'aaa');
  assert(f('aaa', '^.*$') == 'aaa');
  assert(f('aaa', 'b*') == '');
  assert(f('aaa', 'ab*a') == 'aa')
  assert(f('aba', 'ab*a') == 'aba')
  assert(f('aaab', 'a+') == 'aaa')
  assert(f('aaa', '^.+$') == 'aaa')
  assert(not f('aaa', 'b+'))
  assert(not f('aaa', 'ab+a'))
  assert(f('aba', 'ab+a') == 'aba')
  assert(f('a$a', '.$') == 'a')
  assert(f('a$a', '.%$') == 'a$')
  assert(f('a$a', '.$.') == 'a$a')
  assert(not f('a$a', '$$'))
  assert(not f('a$b', 'a$'))
  assert(f('a$a', '$') == '')
  assert(f('', 'b*') == '')
  assert(not f('aaa', 'bb*'))
  assert(f('aaab', 'a-') == '')
  assert(f('aaa', '^.-$') == 'aaa')
  assert(f('aabaaabaaabaaaba', 'b.*b') == 'baaabaaabaaab')
  assert(f('aabaaabaaabaaaba', 'b.-b') == 'baaab')
  assert(f('alo xo', '.o$') == 'xo')
  assert(f(' \n isto \195\169 assim', '%S%S*') == 'isto')
  assert(f(' \n isto \195\169 assim', '%S*$') == 'assim')
  assert(f(' \n isto \195\169 assim', '[a-z]*$') == 'assim')
  assert(f('um caracter ? extra', '[^%sa-z]') == '?')
  assert(f('aa', '^aa?a?a') == 'aa')
  assert(f(']]]\195\161b', '[^]]+') == '\195\161b')
  assert(f("0alo alo", "%x*") == "0a")
  assert(f("alo alo", "%C+") == "alo alo")
end
