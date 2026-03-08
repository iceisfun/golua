-- math.random with seed produces identical sequences to Lua 5.4
math.randomseed(42)
local r = {}
for i = 1, 10 do r[i] = math.random(100) end
assert(table.concat(r, ",") == "50,76,86,54,64,7,25,3,24,24",
    "random(100) sequence mismatch: " .. table.concat(r, ","))

math.randomseed(42)
local r2 = {}
for i = 1, 10 do r2[i] = math.random(1, 1000) end
assert(table.concat(r2, ",") == "742,50,332,342,950,576,263,626,153,259",
    "random(1,1000) sequence mismatch: " .. table.concat(r2, ","))
