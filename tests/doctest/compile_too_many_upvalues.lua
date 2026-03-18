-- A function referencing 256+ upvalues in a long expression should error
-- "too many upvalues", not "too many registers".

local lim = 127
local s = "local function fooA ()\n  local "
for j = 1, lim do
  s = s .. "a" .. j .. ", "
end
s = s .. "b,c\n"
s = s .. "local function fooB ()\n  local "
for j = 1, lim do
  s = s .. "b" .. j .. ", "
end
s = s .. "b\n"
s = s .. "function fooC () return b+c"
local c = 1 + 2
for j = 1, lim do
  s = s .. "+a" .. j .. "+b" .. j
  c = c + 2
end
s = s .. "\nend  end end"
local a, b = load(s)
print(a == nil)
--> =true
print(b:find("too many upvalues") ~= nil)
--> =true
print(b:find("line 5") ~= nil)
--> =true
