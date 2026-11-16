-- A function called through t[k] with a non-constant key is named '?', and
-- one called through an empty-string key is named '' (an empty name, not a
-- missing one). Argument errors, debug.getinfo and tracebacks all use the
-- call-site name and do not fall back to the function's global name.

local function e(f)
  local ok, m = pcall(f)
  return ok, (m:gsub("^[^:]*:%d+: ", ""))
end

local t = {rep = string.rep, [""] = string.rep}
local k = "rep"

print(e(function() t[k]() end))
--> =false	bad argument #1 to '?' (string expected, got no value)

print(e(function() t[""]() end))
--> =false	bad argument #1 to '' (string expected, got no value)

t[""] = function()
  local i = debug.getinfo(1, "n")
  return i.name == "", i.namewhat
end
print(t[""]())
--> =true	field

local function frame(tb)
  return (tb:match("\n\t([^\n]*)"):gsub("^[^:]*:%d+: ", ""))
end

local u = {f = error, [""] = error}
local kf = "f"
print(frame(select(2, xpcall(function() u[kf]("x") end, debug.traceback))))
--> =[C]: in field '?'
print(frame(select(2, xpcall(function() u[""]("x") end, debug.traceback))))
--> =[C]: in field ''

u[""] = function() return debug.traceback("m", 1) end
print(frame(u[""]()))
--> =in field ''

-- A stripped upvalue has no name, so the call is named '?'.
local prog = [[local up = string.rep; return function() up() end]]
local f = assert(load(string.dump(assert(load(prog)), true)))()
print(e(f))
--> =false	bad argument #1 to '?' (string expected, got no value)
