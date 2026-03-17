-- Duplicate label error reports the line of the original label definition.
-- The label's line is recorded as the line of the opening "::" token.
--
-- Note: Lua 5.4 processes adjacent labels recursively (inside-out), which
-- can cause the inner label to be created first. This means for adjacent
-- duplicate labels, lua5.4 may report a different "defined on" line than
-- golua, which processes labels in order.

-- Two labels on consecutive lines: first on line 1, duplicate on line 2.
-- golua stores the first label's line as 1 (the opening :: line).
local f1, e1 = load("::lbl::\n::lbl::")
print(e1:match("on line (%d+)"))
--> =1

-- Same-line labels with intervening label: "::a:: ::b::\n::a::".
-- golua processes in order, so the first ::a:: (line 1) is stored first.
local f2, e2 = load("::a:: ::b::\n::a::")
print(e2:match("on line (%d+)"))
--> =1
