-- Test that function call errors report the line of the call site
-- (the "(" token), not the line where the expression starts.

-- Multi-line call: "a" on line 1, "(" on line 2
local f = load("a\n(23)")
local ok, msg = pcall(f)
local line = tonumber(msg:match(":(%d+):"))
print(line) --> 2

-- For-in iterator error on the line of the iterator expression
-- "for k,v in" on line 3, "3" (iterator) on line 4
f = load("\n\n for k,v in \n 3 \n do \n print(k) \n end")
ok, msg = pcall(f)
line = tonumber(msg:match(":(%d+):"))
print(line) --> 4
