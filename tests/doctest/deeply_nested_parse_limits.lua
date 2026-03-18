-- Deeply nested expressions should hit a parsing depth limit.
-- C Lua uses "C stack overflow" because its recursive-descent parser
-- overflows the C call stack. golua uses the same error message with a
-- counter-based depth limit. Both reject excessively nested expressions.

-- Deeply nested parentheses (200 levels) — both lua5.4 and golua reject
local deep = "return " .. string.rep("(", 200) .. "1" .. string.rep(")", 200)
local f1, e1 = load(deep)
print(f1 == nil)
--> =true
