-- activelines leaves out a vararg function's VARARGPREP, as reference's
-- collectvalidlines does, so the line of the first statement is not listed
-- when that statement's own instructions sit on later lines.
local function lines(f)
  local t = {}
  for l in pairs(debug.getinfo(f, "L").activelines) do t[#t + 1] = l end
  table.sort(t)
  return table.concat(t, ",")
end

local function f(...)
  local function g()
  end
  return g
end
local function h(a)
  local function g()
  end
  return g
end
print(lines(f))
--> =13,14,15
print(lines(h))
--> =18,19,20
print(lines(load("\n\nlocal function q()\nend\nreturn q")))
--> =4,5
print(lines(function(...) return ... end))
--> =27
