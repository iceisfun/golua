-- Test: Missing 'end'/')'/'until' errors should include "(to close 'X' at line N)"
-- when the opening and closing tokens are on different lines.
-- Bug: GoLua's parser has checkMatch() defined but never called for most tokens.

-- Missing 'end' for function
local _, err = load("function f()\n\nreturn 1\n\n")
assert(err:find("to close 'function' at line 1"),
    "function close: got: " .. tostring(err))

-- Missing 'end' for if
_, err = load("if true then\n\nlocal x = 1\n")
assert(err:find("to close 'if' at line 1"),
    "if close: got: " .. tostring(err))

-- Missing 'end' for while
_, err = load("while true do\n\nbreak\n")
assert(err:find("to close 'while' at line 1"),
    "while close: got: " .. tostring(err))

-- Missing 'end' for for
_, err = load("for i = 1, 10 do\n\nlocal x = i\n")
assert(err:find("to close 'for' at line 1"),
    "for close: got: " .. tostring(err))

-- Missing 'end' for do block
_, err = load("do\n\nlocal x = 1\n")
assert(err:find("to close 'do' at line 1"),
    "do close: got: " .. tostring(err))

-- Missing 'until' for repeat
_, err = load("repeat\n\nlocal x = 1\n")
assert(err:find("to close 'repeat' at line 1"),
    "repeat close: got: " .. tostring(err))

print("PASS")
