-- Multiple assignment to table fields must execute right-to-left
-- Bug: GoLua assigned left-to-right, Lua 5.4 assigns right-to-left

-- Overlapping assignment: leftmost wins
do
    local t = {0, 0}
    t[1], t[1] = "first", "second"
    assert(t[1] == "first", "expected 'first', got '" .. tostring(t[1]) .. "'")
end

-- __newindex order: right-to-left
do
    local log = {}
    local t = setmetatable({}, {
        __newindex = function(self, k, v)
            log[#log+1] = k .. "=" .. tostring(v)
            rawset(self, k, v)
        end
    })
    t.a, t.b, t.c = 1, 2, 3
    assert(table.concat(log, ",") == "c=3,b=2,a=1",
        "expected 'c=3,b=2,a=1', got '" .. table.concat(log, ",") .. "'")
end

-- Index-dependent assignment
do
    local t = {1, 0, 0}
    t[1], t[t[1]] = 3, 99
    -- Both target index 1: right-to-left means t[t[1]]=99 first (t[1]=99),
    -- then t[1]=3 overwrites
    assert(t[1] == 3, "expected 3, got " .. tostring(t[1]))
end

