-- Test: Parser error messages match Lua 5.4 wording

-- x += 1 → "syntax error near '+'"
local ok1, err1 = load("x += 1")
assert(not ok1)
assert(string.find(err1, "syntax error near '+'", 1, true),
       "expected 'syntax error near +' in: " .. tostring(err1))

-- print() = 1 → "syntax error near '='"
local ok2, err2 = load("print() = 1")
assert(not ok2)
assert(string.find(err2, "syntax error near '='", 1, true),
       "expected 'syntax error near =' in: " .. tostring(err2))

-- x + 1 as statement → "syntax error near '+'"
local ok3, err3 = load("x + 1")
assert(not ok3)
assert(string.find(err3, "syntax error near '+'", 1, true),
       "expected 'syntax error near +' in: " .. tostring(err3))

-- Valid statements should still work
assert(load("x = 1"))
assert(load("print()"))
assert(load("x, y = 1, 2"))

print("OK")
