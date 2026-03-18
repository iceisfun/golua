-- Duplicate label error reports the line where the label was already defined.
-- When labels are adjacent (no statements between them), Lua 5.4's recursive
-- labelstat() processes inner labels first, so the "defined on" line refers
-- to the later (already-registered) label. golua matches this behavior.

-- Two adjacent labels: second is registered first (reverse order), so the
-- error for the first says "already defined on line 2".
local f1, e1 = load("::lbl::\n::lbl::")
print(e1:match("on line (%d+)"))
--> =2

-- Same-line labels with intervening label: "::a:: ::b::\n::a::".
-- The three labels are adjacent (no real statements between them).
-- Reverse order: ::a:: on line 2 is registered first, then ::b:: on line 1,
-- then ::a:: on line 1 finds duplicate on line 2.
local f2, e2 = load("::a:: ::b::\n::a::")
print(e2:match("on line (%d+)"))
--> =2

-- Non-adjacent labels (real statement between them): first is registered,
-- so the error for the second says "already defined on line 1".
local f3, e3 = load("::lbl::\nlocal x = 1\n::lbl::")
print(e3:match("on line (%d+)"))
--> =1
