local out = {}

local function iter(state, control)
    out[#out + 1] = tostring(state) .. ":" .. tostring(control)
    return nil
end

local function maker()
    return "S", 99
end

for _ in iter, maker() do
end

print(out[1])
--> =S:99
