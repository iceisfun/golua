local mt = {
    __add = function(a, b)
        return setmetatable({ value = a.value + b.value }, mt)
    end,
}

local a = setmetatable({ value = 1 }, mt)
local b = setmetatable({ value = 2 }, mt)

for i = 1, 200000 do
    a = a + b
end
