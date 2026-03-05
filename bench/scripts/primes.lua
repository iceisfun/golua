local n = 100000
local sieve = {}

for i = 2, n do
    sieve[i] = true
end

for i = 2, math.floor(math.sqrt(n)) do
    if sieve[i] then
        for j = i * i, n, i do
            sieve[j] = false
        end
    end
end

local count = 0
for i = 2, n do
    if sieve[i] then
        count = count + 1
    end
end
