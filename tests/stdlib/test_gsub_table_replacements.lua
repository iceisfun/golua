-- Test: pm.lua - string.gsub with table replacements
-- From: pm.lua
-- What: Tests string.gsub with table replacements, including empty tables, false values, position-capture indexed tables, and metatable `__index` fallback.

do
  assert(string.gsub("alo alo", ".", {}) == "alo alo")
  assert(string.gsub("alo alo", "(.)", {a="AA", l=""}) == "AAo AAo")
  assert(string.gsub("alo alo", "(.).", {a="AA", l="K"}) == "AAo AAo")
  assert(string.gsub("alo alo", "((.)(.?))", {al="AA", o=false}) == "AAo AAo")

  assert(string.gsub("alo alo", "().", {'x','yy','zzz'}) == "xyyzzz alo")

  local t = {}; setmetatable(t, {__index = function (t,s) return string.upper(s) end})
  assert(string.gsub("a alo b hi", "%w%w+", t) == "a ALO b HI")
end
