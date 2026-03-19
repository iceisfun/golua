-- Generic for evaluates all explist expressions even though only the first
-- four results seed the iterator state.

local x = 0

local function bump()
    x = x + 10
    return 99
end

for k in next, { 1 }, nil, nil, bump() do
    break
end

print(x)
--> =10
