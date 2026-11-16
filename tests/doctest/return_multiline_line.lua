-- Reference retstat emits the return instruction after the explist is
-- parsed, so a 'return' whose values span lines belongs to the line of the
-- last value's last token, not to the 'return' keyword. A bare 'return'
-- keeps its own line.

local c = setmetatable({}, {__close = function()
    print(debug.getinfo(2, "l").currentline)
end})

-- The __close handler runs from the return instruction.
load("local c = ...\nlocal y <close> = c\nreturn 1,\n  2", "=c")(c)
--> =4

load("local c = ...\nlocal y <close> = c\nreturn {\n}", "=c")(c)
--> =4

load("local c = ...\nlocal y <close> = c\nreturn", "=c")(c)
--> =3

-- A line hook does not step back to the 'return' line.
local lines = {}
local f = load("local a = 1\nreturn a,\n  a + 1", "=c")
debug.sethook(function(_, l)
    if debug.getinfo(2, "S").source == "=c" then lines[#lines + 1] = l end
end, "l")
f()
debug.sethook()
print(table.concat(lines, ","))
--> =1,2,3
