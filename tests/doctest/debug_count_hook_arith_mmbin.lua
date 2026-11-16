-- An arithmetic or bitwise instruction that succeeds on numbers skips the
-- MMBIN* that follows it, as reference's op_arith does, so the count hook
-- does not count that metamethod fallback. It still runs (and counts) when
-- an operand needs a metamethod or string coercion.
local function count(f)
  local n = 0
  debug.sethook(function() n = n + 1 end, "", 1)
  f()
  debug.sethook()
  return n
end
local x, y, z = 0, 3, 2.5
local t = setmetatable({}, {__add = function() return 1 end})

print(count(function() x = x + 1 end))
--> =9
print(count(function() x = x + y end))
--> =10
print(count(function() x = y * z end))
--> =10
print(count(function() x = y / y end))
--> =10
print(count(function() x = y | y end))
--> =10
print(count(function() x = "1" + y end))
--> =11
print(count(function() x = t + y end))
--> =13
print(count(function() for i = 1, 10 do x = x + i end end))
--> =50
