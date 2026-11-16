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

-- A repeat whose body captures a local closes it on exit after reading the
-- condition, so no event goes back to the 'repeat' line.
print(trace("local x = 0\nrepeat\n  local y = x\n  x = x + 1\n  local f = function() return y end\nuntil x >\n  1\nx = 2"))
--> =1 3 4 5 7 3 4 5 7 8
print(trace("local x = 0\nrepeat\n  local y = x\n  local f = function() return y end\nuntil true\nx = 2"))
--> =1 3 4 5 6

-- A block that ends in a label closes its captured locals once the
-- label's closing '::' has been read, so on the label's line.
print(trace("local t = {}\nfor k, v in pairs({1}) do\n  t[k] = function() return v end\n  ::continue::\nend\nt = nil"))
--> =1 2 3 4 2 5 6
print(trace("for i = 1, 2 do\n  local f = function() return i end\n  ::cont::\nend\nlocal z = 1"))
--> =1 2 3 1 2 3 1 5
print(trace("do\n  local x <close> = nil\n  ::a::\nend\nlocal z = 1"))
--> =2 3 5

-- A while body ending in a label jumps back on the label's line: the
-- body's back JMP is emitted after the label's closing '::' is read.
print(trace("local i = 0\nwhile i < 2 do\n  local function g() return i end\n  i = i + 1\n  ::continue::\nend"))
--> =1 2 3 4 5 2 3 4 5 2 6
print(trace("local i = 0\nwhile i < 2 do\n  i = i + 1\n  ::a:: ::b::\n\nend"))
--> =1 2 3 4 2 3 4 2 6
