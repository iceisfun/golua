-- test_pairs_metamethod: pairs() should honor __pairs metamethod in Lua 5.4

local called = 0
local t = setmetatable({}, {
    __pairs = function(self)
        called = called + 1
        local i = 0
        local keys = {"x", "y", "z"}
        return function(_, _)
            i = i + 1
            local k = keys[i]
            if k == nil then return nil end
            return k, i * 10
        end, self, nil
    end
})

local out = {}
for k, v in pairs(t) do
    out[k] = v
end

assert(called == 1, "pairs should call __pairs exactly once")
assert(out.x == 10 and out.y == 20 and out.z == 30, "pairs should iterate values produced by __pairs")
