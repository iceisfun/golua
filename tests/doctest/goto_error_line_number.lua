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
-- The error line prefix is the line of the first statement after the label
local ok5, err5 = load("do\ngoto L\nlocal x = 1\n::L::\nprint(x)\nend")
print(err5)
--> ~]:5: <goto L> at line 2 jumps into the scope of local 'x'

-- goto in inner function: error line prefix is the function's 'end', not chunk EOF
local ok6, err6 = load("local f = function()\n  do ::inner:: end\n  goto inner\nend")
print(err6)
--> ~]:4: no visible label 'inner' for <goto> at line 3

-- goto in main chunk: error line is the line of the last statement
local ok7, err7 = load("do ::inner:: end\ngoto inner\nprint('hi')")
print(err7)
--> ~]:3: no visible label 'inner' for <goto> at line 2
