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

-- Generic for: matches Lua 5.5 (19). Reference reaches the first TFORCALL
-- from TFORPREP, and every TFORLOOP from TFORCALL, with a goto that skips
-- the count hook, and GoLua now leaves them uncounted too. (The count was
-- 42, then 40 after the ADDI in-place fix, then 37, then 25, then 23 since
-- a successful ADDI skips its MMBINI as reference's does.) The per-event
-- lines still differ in one place: on the last iterator call, GoLua counts
-- the JMP after the comparison (reference runs it inline with donextjump),
-- and that JMP leaves the function without the trailing RETURN0 reference
-- emits and counts.
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
    --> =19
end
