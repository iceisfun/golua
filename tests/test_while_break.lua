-- Test while loop with break
local i = 0
while true do
    i = i + 1
    if i == 5 then break end
end

assert(i == 5)
