-- Test numeric for loop with step
local s = 0
for i = 1, 10, 2 do
    s = s + i
end

assert(s == 25)
