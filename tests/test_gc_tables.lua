-- Test garbage collection with many tables
local t = {}
for i = 1, 10000 do
    t[i] = { x = i }
end

collectgarbage()
assert(#t == 10000)
