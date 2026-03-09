-- Test debug.getlocal/setlocal on for-loop internal state variables
--
-- In Lua 5.4, integer for-loops use counter-based internal state:
--   (for state) #1 = current index (same as visible i)
--   (for state) #2 = remaining iterations counter = (limit - init) / step
--   (for state) #3 = step
--   visible i      = copy of current index
--
-- FORLOOP: counter--; index += step; if counter >= 0 then i = index; continue

-- Test 1: counter representation (no preceding locals to simplify indices)
do
    local counters = {}
    for i = 1, 5 do
        local _, counter = debug.getlocal(1, 3) -- (for state) #2 = counter
        counters[#counters+1] = counter
    end
    assert(counters[1] == 4, "counter iter 1: expected 4, got " .. tostring(counters[1]))
    assert(counters[2] == 3, "counter iter 2: expected 3, got " .. tostring(counters[2]))
    assert(counters[3] == 2, "counter iter 3: expected 2, got " .. tostring(counters[3]))
    assert(counters[4] == 1, "counter iter 4: expected 1, got " .. tostring(counters[4]))
    assert(counters[5] == 0, "counter iter 5: expected 0, got " .. tostring(counters[5]))
end

-- Test 2: counter with step > 1
-- for i = 1, 10, 2: counter = (10 - 1) / 2 = 4  (integer division)
do
    local counters = {}
    for i = 1, 10, 2 do
        local _, counter = debug.getlocal(1, 3)
        counters[#counters+1] = counter
    end
    assert(counters[1] == 4, "step2 iter 1: expected 4, got " .. tostring(counters[1]))
    assert(counters[5] == 0, "step2 iter 5: expected 0, got " .. tostring(counters[5]))
end

-- Test 3: counter with negative step
-- for i = 5, 1, -1: counter = (1 - 5) / (-1) = 4
do
    local counters = {}
    for i = 5, 1, -1 do
        local _, counter = debug.getlocal(1, 3)
        counters[#counters+1] = counter
    end
    assert(counters[1] == 4, "neg step iter 1: expected 4, got " .. tostring(counters[1]))
    assert(counters[5] == 0, "neg step iter 5: expected 0, got " .. tostring(counters[5]))
end

-- Test 4: state1 (index) equals visible i
do
    for i = 1, 5 do
        local _, idx = debug.getlocal(1, 1)
        assert(idx == i, "state1 index: expected " .. i .. ", got " .. tostring(idx))
    end
end

-- Test 5: setlocal on index (state1) changes future i values
do
    local out = {}
    for i = 1, 5 do
        out[#out+1] = i
        if i == 2 then
            debug.setlocal(1, 2, 10)  -- set index (state1, but shifted by 'out')
        end
    end
    assert(out[1] == 1, "setidx out[1]=" .. tostring(out[1]))
    assert(out[2] == 2, "setidx out[2]=" .. tostring(out[2]))
    assert(out[3] == 11, "setidx out[3]: expected 11, got " .. tostring(out[3]))
    assert(out[4] == 12, "setidx out[4]: expected 12, got " .. tostring(out[4]))
    assert(out[5] == 13, "setidx out[5]: expected 13, got " .. tostring(out[5]))
end

-- Test 6: setlocal on counter (state2) to 0 stops the loop
do
    local out = {}
    for i = 1, 20 do
        out[#out+1] = i
        if i == 3 then
            debug.setlocal(1, 3, 0)  -- set counter to 0 (shifted by 'out')
        end
    end
    assert(#out == 3, "counter=0: expected 3 elements, got " .. #out)
end

-- Test 7: setlocal on counter to extend the loop
do
    local out = {}
    for i = 1, 5 do
        out[#out+1] = i
        if i == 3 then
            debug.setlocal(1, 3, 10)  -- extend by setting counter to 10
        end
    end
    assert(#out == 13, "counter=10: expected 13 elements, got " .. #out)
end

-- Test 8: setlocal on step (state3) changes increment
do
    local out = {}
    for i = 1, 30 do
        out[#out+1] = i
        if i == 2 then
            debug.setlocal(1, 4, 3)  -- set step to 3 (shifted by 'out')
        end
        if #out > 50 then break end
    end
    assert(out[1] == 1, "step=3 out[1]=" .. tostring(out[1]))
    assert(out[2] == 2, "step=3 out[2]=" .. tostring(out[2]))
    assert(out[3] == 5, "step=3 out[3]: expected 5, got " .. tostring(out[3]))
    assert(out[4] == 8, "step=3 out[4]: expected 8, got " .. tostring(out[4]))
end

print("OK")
