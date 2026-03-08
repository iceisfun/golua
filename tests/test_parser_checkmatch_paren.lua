-- Test: Parser includes "(to close '(' at line N)" for unmatched parentheses

-- Primary expression: (expr
local ok1, err1 = load("(\n1\n")
assert(not ok1)
assert(string.find(err1, "to close '(' at line 1", 1, true),
       "expected 'to close' in: " .. tostring(err1))

-- Function call: f(expr
local ok2, err2 = load("f(\n1\n")
assert(not ok2)
assert(string.find(err2, "to close '(' at line 1", 1, true),
       "expected 'to close' in: " .. tostring(err2))

-- Multi-line function call
local ok3, err3 = load("print(\n1,\n2\n")
assert(not ok3)
assert(string.find(err3, "to close '(' at line 1", 1, true),
       "expected 'to close' in: " .. tostring(err3))

-- Same line should NOT include "to close" (just "')' expected")
local ok4, err4 = load("print(1")
assert(not ok4)
assert(string.find(err4, "')' expected", 1, true),
       "expected ')' expected in: " .. tostring(err4))

print("OK")
