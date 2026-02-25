-- test_ipairs_metamethod: ipairs should use t[i] semantics (honor __index)

local t = setmetatable({}, {
    __index = function(_, k)
        if k >= 1 and k <= 3 then
            return k * 10
        end
        return nil
    end,
})

local seen = {}
for i, v in ipairs(t) do
    seen[#seen + 1] = i
    seen[#seen + 1] = v
end

assert(#seen == 6, "ipairs should iterate 3 elements via __index")
assert(seen[1] == 1 and seen[2] == 10, "first ipairs pair mismatch")
assert(seen[3] == 2 and seen[4] == 20, "second ipairs pair mismatch")
assert(seen[5] == 3 and seen[6] == 30, "third ipairs pair mismatch")
