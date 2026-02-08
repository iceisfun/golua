print("--- Starting Self-Mutation Probe ---")

local function probe()
    local target = {val = 10}

    -- The metamethod captures 'target' as an upvalue
    local mt = {
        __add = function(a, b)
            print("  [Metamethod] Modifying upvalue...")
            target.val = target.val + 20
            -- a and b are both 'target' (same table reference),
            -- so a.val and b.val see the mutation (30 + 30 = 60)
            return a.val + b.val
        end
    }
    setmetatable(target, mt)

    -- Trigger the add.
    -- The VM has target in a register, then calls the metamethod.
    local result = target + target

    print("  Result of target + target:", result)
    print("  New value of target.val:", target.val)

    -- 60 is correct: a and b are the same table, mutation is visible through both
    assert(result == 60, "Expected 60, got " .. tostring(result))
    assert(target.val == 30, "Upvalue mutation failed")
end

local ok, err = pcall(probe)
if ok then
    print("PASS: VM survived self-mutating metamethods")
else
    print("!! Failure: " .. tostring(err))
end

--------------------------------------------------------------------------------
-- Rug-Pull Probe: force rehash + GC during metamethod execution
--------------------------------------------------------------------------------

print("\n--- Starting Rug-Pull Probe ---")

local function rug_pull_probe()
    local target = {val = 1}
    setmetatable(target, {
        __add = function(a, b)
            print("  [Metamethod] Forcing rehash and GC...")
            -- Force a massive rehash of the object being operated on
            for i = 1, 1000 do target["key"..i] = i end
            -- Force a GC cycle (if available)
            if collectgarbage then collectgarbage("collect") end
            return a.val + b.val
        end
    })

    -- If the VM holds a stale/unsafe pointer to 'target', this could panic
    local result = target + target
    print("  Result:", result)
    -- a.val and b.val are both 1 (val field unchanged), so result = 2
    assert(result == 2, "Expected 2, got " .. tostring(result))
end

local ok2, err2 = pcall(rug_pull_probe)
if ok2 then
    print("PASS: VM survived rug-pull")
else
    print("!! Crash/Error:", err2)
end

--------------------------------------------------------------------------------
-- Type-Swap Probe: change value type during metamethod
--------------------------------------------------------------------------------

print("\n--- Starting Type-Swap Probe ---")

local function type_swap_probe()
    local target = {val = 10}
    local call_count = 0
    setmetatable(target, {
        __add = function(a, b)
            call_count = call_count + 1
            print("  [Metamethod] Call #" .. call_count .. ", swapping val type...")
            -- Change val from number to string
            target.val = "swapped"
            -- But return a numeric result
            return 42
        end
    })

    local result = target + target
    print("  Result:", result)
    assert(result == 42, "Expected 42, got " .. tostring(result))
    assert(target.val == "swapped", "Type swap failed")
    assert(call_count == 1, "Metamethod called wrong number of times")
end

local ok3, err3 = pcall(type_swap_probe)
if ok3 then
    print("PASS: VM survived type-swap")
else
    print("!! Crash/Error:", err3)
end

--------------------------------------------------------------------------------
-- Metatable-Swap Probe: replace metatable during metamethod
--------------------------------------------------------------------------------

print("\n--- Starting Metatable-Swap Probe ---")

local function mt_swap_probe()
    local target = {val = 5}
    local original_mt
    original_mt = {
        __add = function(a, b)
            print("  [Metamethod] Replacing own metatable...")
            -- Replace the metatable mid-operation
            setmetatable(target, {
                __add = function(x, y)
                    return 999
                end
            })
            return a.val + b.val
        end
    }
    setmetatable(target, original_mt)

    -- First call uses original metamethod
    local r1 = target + target
    print("  First result:", r1)
    assert(r1 == 10, "Expected 10, got " .. tostring(r1))

    -- Second call should use the new metamethod
    local r2 = target + target
    print("  Second result:", r2)
    assert(r2 == 999, "Expected 999, got " .. tostring(r2))
end

local ok4, err4 = pcall(mt_swap_probe)
if ok4 then
    print("PASS: VM survived metatable-swap")
else
    print("!! Crash/Error:", err4)
end

print("\n--- All Probes Complete ---")
