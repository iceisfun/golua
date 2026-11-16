-- A native function used as a metamethod is named after the metamethod in
-- its argument errors, whichever instruction dispatched it: __index and
-- __newindex through an upvalue table (OP_GETTABUP / OP_SETTABUP) or a
-- method call (OP_SELF), and __close from OP_CLOSE / OP_RETURN. A close run
-- while unwinding an error, or a metamethod called from a C function, has
-- no caller name and falls back to the global name.

local function e(f, ...)
  local ok, m = pcall(f, ...)
  return ok, (m:gsub("^[^:]*:%d+: ", ""))
end

local u = setmetatable({}, {__index = string.rep, __newindex = string.rep})

print(e(function() u.x = 1 end))
--> =false	bad argument #1 to 'newindex' (string expected, got table)

print(e(function() return u:x() end))
--> =false	bad argument #1 to 'index' (string expected, got table)

local function closer(f)
  return setmetatable({}, {__close = f})
end

print(e(function() local x <close> = closer(string.rep) end))
--> =false	bad argument #1 to 'close' (string expected, got table)

print(e(function() do local x <close> = closer(load) end return 1 end))
--> =false	bad argument #1 to 'close' (function expected, got table)

print(e(function() local x <close> = closer(string.rep) error("e") end))
--> =false	bad argument #1 to 'string.rep' (string expected, got table)

print(e(table.unpack, setmetatable({}, {__index = string.rep}), 1, 1))
--> =false	bad argument #1 to 'string.rep' (string expected, got table)

local pcall, print, e = pcall, print, e
_ENV = setmetatable({}, {__newindex = string.rep, __index = _G})
print(e(function() zz = 1 end))
--> =false	bad argument #1 to 'newindex' (string expected, got table)
