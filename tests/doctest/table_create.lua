-- table.create(narr [, nrec]) preallocates a table with capacity hints.
-- The returned table is always empty regardless of the capacity hints.

-- Array slots only
local t2 = table.create(10)
print(type(t2))
--> =table
print(#t2)
--> =0

-- Array and hash slots
local t3 = table.create(10, 5)
print(type(t3))
--> =table
print(#t3)
--> =0

-- The table works normally: set and get values
t3[1] = "hello"
t3[2] = "world"
t3.key = "value"
print(t3[1])
--> =hello
print(t3[2])
--> =world
print(t3.key)
--> =value
print(#t3)
--> =2

-- table.create(0, 0) is valid
local t4 = table.create(0, 0)
print(type(t4))
--> =table
print(#t4)
--> =0

-- table.create(0) is valid (nrec defaults to 0)
local t5 = table.create(0)
print(type(t5))
--> =table
print(#t5)
--> =0
