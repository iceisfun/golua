-- Line attribution of the instructions a loop runs on its way out, matched
-- against reference Lua's line-hook traces.
local function trace(src)
  local f, t = assert(load(src)), {}
  debug.sethook(function(_, l)
    if debug.getinfo(2, "S").source == src then t[#t + 1] = l end
  end, "l")
  f()
  debug.sethook()
  return table.concat(t, " ")
end

-- A generic for closes its iterator's closing value after reading 'end',
-- so that close is on the 'end' line; TFORLOOP is on the explist line like
-- TFORCALL, and TFORPREP on the 'do' line.
print(trace("local x\nfor k, v in pairs({1}) do\n  x = v\nend\nx = 2"))
--> =1 2 3 2 4 5
print(trace("local x\nfor k, v in pairs({1}) do\n  goto cont\n  ::cont::\nend\nx = 2"))
--> =1 2 3 2 5 6
print(trace("for k, v in\n  pairs({1}) do\nend\nlocal z = 1"))
--> =2 2 3 4
print(pcall(load("for k in next, {},\n  nil, 1\n  do end", "=forin")))
--> =false	forin:3: variable '(for state)' got a non-closable value

local function iter() return nil end
for k in iter, nil, nil, setmetatable({}, {__close = function()
  print(debug.getinfo(2, "l").currentline)
end}) do
end
--> =29
