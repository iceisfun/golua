-- Test: pm.lua - string.gsub with string replacements
-- From: pm.lua
-- What: Tests string.gsub with literal replacements, capture references (`%1`, `%0`), position captures, trimming, and replacement count limits.

do
  assert(string.gsub('\195\188lo \195\188lo', '\195\188', 'x') == 'xlo xlo')
  assert(string.gsub('alo \195\186lo  ', ' +$', '') == 'alo \195\186lo')  -- trim
  assert(string.gsub('  alo alo  ', '^%s*(.-)%s*$', '%1') == 'alo alo')  -- double trim
  assert(string.gsub('alo  alo  \n 123\n ', '%s+', ' ') == 'alo alo 123 ')
  assert(string.gsub('alo alo', '()[al]', '%1') == '12o 56o')
  assert(string.gsub("abc=xyz", "(%w*)(%p)(%w+)", "%3%2%1-%0") ==
                "xyz=abc-abc=xyz")
  assert(string.gsub("abc", "%w", "%1%0") == "aabbcc")
  assert(string.gsub("abc", "%w+", "%0%1") == "abcabc")
  assert(string.gsub('\195\161\195\169\195\173', '$', '\0\195\179\195\186') == '\195\161\195\169\195\173\0\195\179\195\186')
  assert(string.gsub('', '^', 'r') == 'r')
  assert(string.gsub('', '$', 'r') == 'r')
end
