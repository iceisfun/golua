-- goto "no visible label" errors should report the EOF line number as the
-- error location prefix, matching Lua 5.4 behavior.

-- Single-line: goto to undefined label, EOF is line 1
local ok, err = load("goto L")
print(err)
--> ~]:1: no visible label 'L' for <goto> at line 1

-- Multi-line: the error prefix line should be the last line (EOF), not the goto line
local ok2, err2 = load("goto L\nprint(1)\nprint(2)")
print(err2)
--> ~]:3: no visible label 'L' for <goto> at line 1

-- goto into nested block: label inside do-end not visible from outside
local ok3, err3 = load("goto L\ndo\n::L::\nend")
print(err3)
--> ~]:4: no visible label 'L' for <goto> at line 1

-- goto across function boundary: label in inner function not visible
local ok4, err4 = load("local function f()\n::L::\nend\ngoto L")
print(err4)
--> ~]:4: no visible label 'L' for <goto> at line 4

-- goto jumping into scope of local variable (compile error)
-- Reference Lua raises this at leaveblock, so the error line prefix is
-- ls->lastline: the end line of the block's LAST statement. Here that is
-- print(x) on line 5 (also the only statement after the label).
local ok5, err5 = load("do\ngoto L\nlocal x = 1\n::L::\nprint(x)\nend")
print(err5)
--> ~]:5: <goto L> at line 2 jumps into the scope of 'x'

-- When several statements follow the label, the prefix is the LAST one's
-- line (here line 5, 'local c'), NOT the first statement after the label
-- (line 4) nor the 'end' keyword (line 6). Verified against lua5.5.0.
local ok5b, err5b = load("do\ngoto L\nlocal x\n::L::\nlocal a\nlocal c\nend")
print(err5b)
--> ~]:6: <goto L> at line 2 jumps into the scope of 'x'

-- function body: close_func runs leaveblock AFTER the closing 'end' is
-- consumed, so unlike a do/while/for block the prefix is the 'end' line (6),
-- NOT the body's last statement (line 5). Verified against lua5.5.0.
local ok5d, err5d = load("local function f()\ngoto L\nlocal x\n::L::\nlocal y\nend")
print(err5d)
--> ~]:6: <goto L> at line 2 jumps into the scope of 'x'

-- A multi-line trailing expression as the block's last statement: the prefix
-- is the line of the statement's FINAL token (the closing ')' on line 7), not
-- the statement's start line. Exercises AST End()/span tracking.
local ok5e, err5e = load("do\ngoto L\nlocal x\n::L::\nprint(\n1\n)\nend")
print(err5e)
--> ~]:7: <goto L> at line 2 jumps into the scope of 'x'

-- repeat-until: body locals stay visible in the condition, so the goto is
-- resolved only after 'until', and the prefix is the condition's line (5).
local ok5c, err5c = load("repeat\ngoto L\nlocal x\n::L::\nuntil false")
print(err5c)
--> ~]:5: <goto L> at line 2 jumps into the scope of 'x'

-- goto in inner function: error line prefix is the function's 'end', not chunk EOF
local ok6, err6 = load("local f = function()\n  do ::inner:: end\n  goto inner\nend")
print(err6)
--> ~]:4: no visible label 'inner' for <goto> at line 3

-- goto in main chunk: error line is the line of the last statement
local ok7, err7 = load("do ::inner:: end\ngoto inner\nprint('hi')")
print(err7)
--> ~]:3: no visible label 'inner' for <goto> at line 2
