-- Vararg function activelines should not include the definition line
local f = load("return function(...)\n  local a = 20\nend")()
local info = debug.getinfo(f, "L")
local lines = {}
for k in pairs(info.activelines) do lines[#lines+1] = k end
table.sort(lines)
print(table.concat(lines, ","))
--> 2,3
