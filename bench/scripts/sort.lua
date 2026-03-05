math.randomseed(12345)

local t = {}
for i = 1, 20000 do
    t[i] = math.random()
end

table.sort(t)
