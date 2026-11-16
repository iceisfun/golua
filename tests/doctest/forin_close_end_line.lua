-- A generic for closes its closing variable when the loop ends. Reference
-- Lua emits that OP_CLOSE in forstat's leaveblock, after 'end' is consumed,
-- so it belongs to the 'end' line, not the 'for' header line. That line is
-- what a __close handler sees as its caller's currentline, what a line hook
-- reports when the loop ends, and it is listed in activelines.

local c = setmetatable({}, {__close = function()
    print(debug.getinfo(2, "l").currentline)
end})

load("local c = ...\nfor k in next, {1}, nil, c do\n  local z = k\nend", "=c")(c)
--> =4

-- A break closes the loop on its own line.
load("local c = ...\nfor k in next, {1}, nil, c do\n  break\nend", "=c")(c)
--> =3

local function active(f)
    local t = {}
    for l in pairs(debug.getinfo(f, "L").activelines) do t[#t + 1] = l end
    table.sort(t)
    return table.concat(t, ",")
end

print(active(load("for k in pairs({}) do\nend\nreturn 1", "=c")))
--> =1,2,3

print(active(load("return function()\n  for k in pairs({}) do\n  end\nend", "=c")()))
--> =2,3,4

local lines = {}
local f = load("local c = ...\nfor k in next, {1}, nil, c do\n  local z = k\nend\nlocal y = 1", "=c")
debug.sethook(function(_, l)
    if debug.getinfo(2, "S").source == "=c" then lines[#lines + 1] = l end
end, "l")
f(setmetatable({}, {__close = function() end}))
debug.sethook()
print(table.concat(lines, ","))
--> =1,2,3,2,4,5
