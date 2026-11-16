-- A multi-line table constructor's final SETLIST, and the discharge of its
-- last item, carry the line of the closing '}' as in reference Lua (which
-- emits them once the '}' has been read), so the line hook does not jump
-- back to the '{' line. A mid-constructor flush carries the flushed item's
-- line.
local function trace(src)
  local f, t = assert(load(src)), {}
  debug.sethook(function(_, l)
    if debug.getinfo(2, "S").what == "main" then t[#t + 1] = l end
  end, "l")
  f()
  debug.sethook()
  return table.concat(t, " ")
end

print(trace("local t = {1,\n  2}"))
--> =1 2
print(trace("local t = {\n  1,\n  2,\n}"))
--> =1 2 4
print(trace("local t = {\n  1,\n  function()\n  end,\n  3\n}"))
--> =1 2 4 6
print(trace("local t = {\n  print\n}"))
--> =1 3
print(trace("local t = {\n  1,\n  x = 2\n}"))
--> =1 2 3 4

local s = {"local t = {"}
for i = 1, 52 do s[#s + 1] = "  " .. i .. "," end
s[#s + 1] = "}"
print((trace(table.concat(s, "\n")):gsub("^1 2 .* 51 ", "")))
--> =52 54

-- runtime errors in the last item report the line it is discharged on
print(pcall(load("local t = {\n  a.b\n}", "=ctor")))
--> =false	ctor:3: attempt to index a nil value (global 'a')
