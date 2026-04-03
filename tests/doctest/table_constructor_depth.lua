-- Test dynamic maxtostore for table constructors (Lua 5.5 feature).
-- The dynamic threshold replaces the fixed LFIELDS_PER_FLUSH=50, allowing
-- deeper nesting of table constructors with many array items.

-- Basic constructor still works.
print(type({1,2,3}))
--> =table

-- Constructor with >50 array items still works.
local t = {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,
           21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
           41,42,43,44,45,46,47,48,49,50,51}
print(#t, t[1], t[51])
--> =51	1	51

-- Deep nesting with many array items per level.
-- With a fixed threshold of 50, this would exhaust registers because each
-- level needs 45 + overhead registers. The dynamic threshold flushes sooner
-- when fewer registers are available, allowing deeper recursion.
local function deep(n)
  if n == 0 then return {} end
  local inner = deep(n - 1)
  local t = {1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,
             21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
             41,42,43,44,45,inner}
  return t
end

print(type(deep(5)))
--> =table

print(type(deep(6)))
--> =table

-- Verify the nested structure is correct.
local d = deep(3)
print(#d, d[45], type(d[46]))
--> =46	45	table
print(#d[46], d[46][45], type(d[46][46]))
--> =46	45	table
print(#d[46][46], d[46][46][45], type(d[46][46][46]))
--> =46	45	table
