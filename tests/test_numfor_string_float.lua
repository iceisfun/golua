-- Numeric for loop with string parameters should force float conversion
-- In Lua 5.4, when the initial value or step of a numeric for loop
-- is a string, all for-loop variables are converted to floats.
-- GoLua incorrectly keeps them as integers when the string represents
-- an integer value.

-- String init forces float
local t1 = {}
for i = "1", 3, 1 do
  t1[#t1+1] = math.type(i)
end
assert(t1[1] == "float",
  "for i='1',3,1: i should be float, got " .. t1[1])

-- String step forces float
local t2 = {}
for i = 1, 3, "1" do
  t2[#t2+1] = math.type(i)
end
assert(t2[1] == "float",
  "for i=1,3,'1': i should be float, got " .. t2[1])

-- All strings forces float
local t3 = {}
for i = "1", "3", "1" do
  t3[#t3+1] = math.type(i)
end
assert(t3[1] == "float",
  "for i='1','3','1': i should be float, got " .. t3[1])

-- String init and int limit+step forces float
local t4 = {}
for i = "1", 5, 2 do
  t4[#t4+1] = {i, math.type(i)}
end
assert(t4[1][2] == "float",
  "for i='1',5,2: i should be float, got " .. t4[1][2])
assert(t4[1][1] == 1.0, "value should be 1.0")

-- Hex string init forces float
local t5 = {}
for i = "0x1", 3 do
  t5[#t5+1] = math.type(i)
end
assert(t5[1] == "float",
  "for i='0x1',3: i should be float, got " .. t5[1])

-- String limit only should NOT force float (init and step determine type)
local t6 = {}
for i = 1, "3" do
  t6[#t6+1] = math.type(i)
end
assert(t6[1] == "integer",
  "for i=1,'3': i should be integer, got " .. t6[1])

print("OK")
