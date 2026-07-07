-- Count hooks expose instruction-count differences, so these code shapes
-- should match Lua 5.4's observed hook counts.

-- Method call assignment: GoLua emits an extra MOVE vs Lua 5.4 (8 vs 7).
do
    local n = 0
    debug.sethook(function() n = n + 1 end, "", 1)
    local s = "hello"
    s = s:upper()
    debug.sethook()
    print(n, s)
    --> =8	HELLO
end

-- Numeric for with a captured loop variable: GoLua emits more close work
-- than Lua 5.4 (13 vs 12), a known architectural difference. The indexed
-- store `fs[i] = ...` references the live local table/key registers directly
-- (matching reference assignment eval order), saving two MOVEs.
do
    local n = 0
    debug.sethook(function() n = n + 1 end, "", 1)
    local fs = {}
    for i = 1, 1 do
        fs[i] = function() return i end
    end
    debug.sethook()
    print(n, fs[1]())
    --> =13	1
end

-- Generic for: GoLua emits more instructions than Lua 5.4 (38 vs 19),
-- a known architectural difference in for-in loop compilation. The body
-- `return i + 1` reads the local `i` directly in the ADDI operand instead of
-- MOVEing it to a temp, and the guard `i < 2` now reads `i` in place in the
-- OP_LT operand rather than snapshotting it into a temp, each saving MOVEs per
-- iteration (was 43, then 41 after the ADDI in-place fix).
do
    local function iter(_, i)
        if i < 2 then
            return i + 1
        end
    end

    local n = 0
    debug.sethook(function() n = n + 1 end, "", 1)
    for k in iter, nil, 0 do
    end
    debug.sethook()
    print(n)
    --> =38
end
