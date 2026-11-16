-- A store into a field or index is emitted after its right-hand side is
-- parsed, so reference Lua stamps it with the line of the last token of the
-- RHS: the 'end' of a function, the '}' of a table, the ')' of a call or a
-- parenthesized expression, or the closing bracket of a long string. A
-- runtime error in the store reports that line, not the line the RHS starts.

local function run(src)
    print(pcall(load(src, "=c")))
end

run("local M\nM.handler = function()\n  return 1\nend")
--> =false	c:4: attempt to index a nil value (local 'M')

run("local M\nM.opts = {\n  a = 1,\n}")
--> =false	c:4: attempt to index a nil value (local 'M')

run("local M\nM.v = tostring(\n  1\n)")
--> =false	c:4: attempt to index a nil value (local 'M')

run("local M\nM[1] = {\n}")
--> =false	c:3: attempt to index a nil value (local 'M')

run("local M\nM.x = [[long\nstring]]")
--> =false	c:3: attempt to index a nil value (local 'M')

run("local M\nM.x = 'a\\\nb'")
--> =false	c:3: attempt to index a nil value (local 'M')

run("local M\nM.x = (function()\nend)")
--> =false	c:3: attempt to index a nil value (local 'M')

run("local M\nM.x = ('a'):rep(\n  2\n)")
--> =false	c:4: attempt to index a nil value (local 'M')

run("local M, a\na, M.y = 1, {\n}")
--> =false	c:3: attempt to index a nil value (local 'M')

-- \r\n counts as one line break inside a long string.
run("local M\r\nM.x = [[a\r\nb\r\nc]]")
--> =false	c:4: attempt to index a nil value (local 'M')

-- A comparison is emitted after its right operand too.
run("return 1 < {\n}")
--> =false	c:2: attempt to compare number with table

-- A __newindex handler sees the same line for its caller.
local t = setmetatable({}, {__newindex = function()
    print(debug.getinfo(2, "l").currentline)
end})
load("local t = ...\nt.x = {\n}", "=c")(t)
--> =3
