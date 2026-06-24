-- A long run of always-empty quantifiers must not trip the "pattern too
-- complex" recursion guard. Lua folds a zero-repetition '*'/'-'/'?' in the same
-- match frame (`p = ep + 1; goto init`) rather than recursing, so chains far
-- longer than the recursion limit still match. Previously golua recursed once
-- per quantifier and raised "pattern too complex" past ~200 elements.

print((pcall(string.match, "", string.rep("a*", 250))))
--> =true

print((pcall(string.match, "", string.rep("a-", 250))))
--> =true

print((pcall(string.match, "", string.rep("a?", 250))))
--> =true

-- Prefix consume then a long zero-match tail still works.
print(string.match("aaaa", string.rep("b*", 250) .. "a"))
--> =a

-- Ordinary greedy/lazy matching is unaffected.
print(string.match("aaa", "a*"))
--> =aaa
print(("  trim  "):match("^%s*(.-)%s*$"))
--> =trim
print(string.gsub("hello world", "o", "0"))
--> =hell0 w0rld	2
