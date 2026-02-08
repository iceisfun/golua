-- Test generic for with iterator function
local function iter(t)
    local i = 0
    return function()
        i = i + 1
        return t[i]
    end
end

local t = { 1, 2, 3 }
local out = {}

for v in iter(t) do
    out[#out + 1] = v
end

assert(#out == 3)
assert(out[1] == 1 and out[3] == 3)
