-- test_register_alloc: multi-return comparison corruption and register allocation

-- Multi-return into locals with comparison should not corrupt registers
do
    local function f()
        return 4, "hello", 99
    end

    local total = 0
    local done = false

    while not done do
        local idx, val, ok = f()
        if idx == 4 then
            done = true
        end
        total = ok
    end
    assert(total == 99, "multi-return comparison corruption: expected 99, got " .. tostring(total))
end

-- Same with false comparison
do
    local function f()
        return 5, "hello", 99
    end

    local result = -1
    while true do
        local idx, val, ok = f()
        if idx == 4 then
            -- won't execute
        end
        result = ok
        break
    end
    assert(result == 99, "multi-return false comparison: expected 99, got " .. tostring(result))
end

-- Compound boolean logic on multi-return locals
do
    local function f()
        return 1, 2, 42
    end

    local function test()
        local x, y, z = f()
        if x ~= nil and x > 0 then
            return z
        end
        return -1
    end
    assert(test() == 42, "multi-return boolean expression")
end

-- Nested expression pressure
do
    local function f()
        return 1, 2, 3
    end

    local function test()
        local a, b, c = f()
        if (a and (b or c)) then
            return b
        end
        return -1
    end
    assert(test() == 2, "nested expression pressure")
end

-- Control test (no multi-return)
do
    local function test()
        local a = 1
        local b = 2
        local c = 3
        if a == 1 then
            return c
        end
        return -1
    end
    assert(test() == 3, "control no multi-return")
end

-- For loop comparison corruption
do
    local function f()
        return 1, "data", 77
    end

    local sum = 0
    for i = 1, 3 do
        local idx, val, ok = f()
        if idx == 1 then
            sum = sum + ok
        end
    end
    assert(sum == 231, "for loop comparison: expected 231, got " .. tostring(sum))
end

-- Four return values
do
    local function f4()
        return 1, 2, 3, 100
    end

    local result = -1
    while true do
        local a, b, c, d = f4()
        if a == 1 then
            result = d
        end
        break
    end
    assert(result == 100, "four return values")
end

-- Register top after leave scope
do
    local result = -1
    while true do
        local a = 42
        do
            local b = 99
        end
        if a == 42 then
            result = a
        end
        break
    end
    assert(result == 42, "reg top after leave scope")
end
