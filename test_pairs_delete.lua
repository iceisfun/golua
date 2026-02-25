local t = {}
t.A = 1
t.B = 2
t.C = 3
t.D = 4

local count = 0
for k, v in pairs(t) do
    print("visiting", k)
    count = count + 1
    if k == "B" then
        t.B = nil
    end
end
print("count", count)
