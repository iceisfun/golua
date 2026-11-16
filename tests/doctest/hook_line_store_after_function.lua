-- A store whose value is a function body or a table constructor is emitted
-- after the closing 'end' or '}' has been read, so like reference Lua it
-- carries that line: the line hook does not jump back to the assignment's
-- first line, and an error in the store reports the closing line.
local function trace(src)
  local f, t = assert(load(src)), {}
  debug.sethook(function(_, l)
    if debug.getinfo(2, "S").source == src then t[#t + 1] = l end
  end, "l")
  f()
  debug.sethook()
  return table.concat(t, " ")
end

print(trace("local t = {}\nt.a = function()\n  return 1\nend\nt = nil"))
--> =1 4 5
print(trace("local t = {}\nt[1] = function()\n  return 1\nend\nt = nil"))
--> =1 4 5
print(trace("T = function()\n  return 1\nend\nT = nil"))
--> =3 4
print(trace("local t = {}\nt.a, t.b = 1, function()\n  return 1\nend\nt = nil"))
--> =1 2 4 5
print(trace("local t = {}\nt.a = {\n  1}\nt = nil"))
--> =1 2 3 4

print(pcall(load("local t\nt.a = function()\nend", "=store")))
--> =false	store:3: attempt to index a nil value (local 't')
