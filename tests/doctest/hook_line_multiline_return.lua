-- The RETURN of a multi-line return statement carries the line of the last
-- token of its expression list, as in reference Lua, so the line hook
-- reaches that line instead of jumping back to the 'return' line.
local function trace(src)
  local f, t = assert(load(src)), {}
  debug.sethook(function(_, l)
    if debug.getinfo(2, "S").source == src then t[#t + 1] = l end
  end, "l")
  f()
  debug.sethook()
  return table.concat(t, " ")
end

print(trace("local function f(a, b)\n  return a +\n    b\nend\nlocal r = f(1, 2)"))
--> =4 5 2 3
print(trace("local function f(a, b)\n  return a,\n    b\nend\nlocal r = f(1, 2)"))
--> =4 5 2 3
print(trace("local function f(a)\n  return\n    a\nend\nlocal r = f(1)"))
--> =4 5 3
print(trace("local function f(a)\n  return 'x' ..\n  'y'\nend\nlocal r = f(1)"))
--> =4 5 2 3 2 3

-- a __close handler run by the return sees that line as current
local function g()
  local c <close> = setmetatable({}, {__close = function()
    print(debug.getinfo(2, "l").currentline)
  end})
  return 1,
    2
end
g()
--> =29
