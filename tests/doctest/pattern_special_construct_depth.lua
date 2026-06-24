-- The %f (frontier), %b (balanced) and %1-%9 (back-reference) constructs must
-- fold in the same match frame on success (Lua's `goto init`), not recurse, so
-- a long run of them does not falsely trip the "pattern too complex" recursion
-- guard. Companion to pattern_empty_quantifier_chain.lua, which covers the
-- */-/? quantifier fold; this covers the %-construct fold.

print((pcall(string.match, "abc", string.rep("%f[%a]", 250))))
--> =true

print((pcall(string.match, "", string.rep("%f[%a]", 250))))
--> =true

-- Ordinary frontier/balanced/back-ref matching is unaffected.
print(string.match("abc def", string.rep("%f[%a]", 8) .. "%a+"))
--> =abc
print(string.match("(((x)))", "%b()"))
--> =(((x)))
print(("a1b2"):gsub("(%a)(%d)", "%2%1"))
--> =1a2b	2
print(string.find("word1 word2", "%f[%w]%w+"))
--> =1	5
