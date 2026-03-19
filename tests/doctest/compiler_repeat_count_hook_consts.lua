-- Repeat/until with constant conditions should compile without the extra
-- boolean materialization chain that inflates count-hook-visible work.

do
    local n = 0
    local function hk()
        n = n + 1
    end
    local function f()
        repeat until true
    end
    debug.sethook(hk, "", 1)
    f()
    debug.sethook()
    print(n)
    --> =6
end

do
    local n = 0
    local function hk()
        n = n + 1
    end
    local function f()
        repeat
            local y = 1
        until true
    end
    debug.sethook(hk, "", 1)
    f()
    debug.sethook()
    print(n)
    --> =7
end
