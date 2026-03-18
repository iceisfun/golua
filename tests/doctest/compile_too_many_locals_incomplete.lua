-- Incomplete source with 301 locals should error "too many local variables"
-- even without a closing 'end'. The parser must check the limit during parsing,
-- not wait for the compiler.

local s = "\nfunction foo ()\n  local "
for j = 1, 300 do
  s = s .. "a" .. j .. ", "
end
s = s .. "b\n"
local a, b = load(s)
print(a == nil)
--> =true
print(b:find("too many local variables") ~= nil)
--> =true
print(b:find("line 2") ~= nil)
--> =true
