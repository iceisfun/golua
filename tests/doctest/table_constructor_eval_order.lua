-- Table constructor evaluation order: NEWTABLE must not clobber the
-- target register before RHS expressions (like table.unpack) read it.

-- Basic self-assignment via unpack
local t = {10, 20, 30}
t = {table.unpack(t)}
print(#t, t[1], t[2], t[3])
--> =3	10	20	30

-- Self-assignment with function call in constructor
local a = {1, 2, 3, 4, 5}
a = {table.unpack(a)}
print(#a)
--> =5

-- Constructor with mixed expressions referencing target
local x = {100}
x = {x[1] + 1, x[1] + 2}
print(x[1], x[2])
--> =101	102

-- Nested self-reference
local m = {a = 1, b = 2}
m = {a = m.a + 10, b = m.b + 20}
print(m.a, m.b)
--> =11	22

-- Multi-value unpack preserves all values
local big = {}
for i = 1, 50 do big[i] = i end
big = {table.unpack(big)}
print(#big, big[1], big[50])
--> =50	1	50
