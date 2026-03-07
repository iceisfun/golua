-- Stress test: self-mutation during metamethod execution

-- 1. Upvalue mutation during __add
do
    local target = {val = 10}
    setmetatable(target, {
        __add = function(a, b)
            target.val = target.val + 20
            return a.val + b.val
        end
    })
    local result = target + target
    assert(result == 60, "Expected 60, got " .. tostring(result))
    assert(target.val == 30, "Upvalue mutation failed")
end

-- 2. Rug-pull: force rehash + GC during metamethod
do
    local target = {val = 1}
    setmetatable(target, {
        __add = function(a, b)
            for i = 1, 1000 do target["key"..i] = i end
            collectgarbage("collect")
            return a.val + b.val
        end
    })
    local result = target + target
    assert(result == 2, "Expected 2, got " .. tostring(result))
end

-- 3. Type-swap: change value type during metamethod
do
    local target = {val = 10}
    local call_count = 0
    setmetatable(target, {
        __add = function(a, b)
            call_count = call_count + 1
            target.val = "swapped"
            return 42
        end
    })
    local result = target + target
    assert(result == 42, "Expected 42, got " .. tostring(result))
    assert(target.val == "swapped", "Type swap failed")
    assert(call_count == 1, "Metamethod called wrong number of times")
end

-- 4. Metatable-swap: replace metatable during metamethod
do
    local target = {val = 5}
    setmetatable(target, {
        __add = function(a, b)
            setmetatable(target, {
                __add = function(x, y) return 999 end
            })
            return a.val + b.val
        end
    })
    local r1 = target + target
    assert(r1 == 10, "Expected 10, got " .. tostring(r1))
    local r2 = target + target
    assert(r2 == 999, "Expected 999, got " .. tostring(r2))
end
