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

-- Generic for: GoLua emits more instructions than Lua 5.4 (42 vs 19),
-- a known architectural difference in for-in loop compilation. (Lua 5.5
-- folds the to-be-closed setup into TFORPREP, so GoLua no longer emits a
-- separate OP_TBC instruction here.)
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
    --> =42
end
