-- Some general checks
do
    local function cof(x)
        print("in cof", x)
        print("in cof", coroutine.yield(x + 2))
        return "from cof"
    end

    local co = coroutine.create(cof)

    print(pcall(coroutine.resume, {}))
    --> ~false\t.*thread expected

    print("out", coroutine.resume(co, 1))
    print("out", coroutine.resume(co, "two"))

    --> =in cof	1
    --> =out	true	3
    --> =in cof	two
    --> =out	true	from cof

    print(coroutine.status(co))
    --> =dead

    print(pcall(coroutine.status, cof))
    --> ~false\t.*thread expected

    print(pcall(coroutine.create, 1))
    --> ~false\t.*function expected
end

-- Check the main coroutine is correctly identified
print(coroutine.running())
--> ~^thread:.*\ttrue$

do
    -- Check the main coroutine is not yieldable
    print(coroutine.isyieldable())
    --> =false

    print(pcall(coroutine.yield, 1))
    --> ~false\t.*yield from outside a coroutine

    -- isyieldable(non-thread) errors like Lua 5.4
    print(pcall(coroutine.isyieldable, 1))
    --> ~false\t.*thread expected, got number
end

-- Check that coroutine.running() returns false as second argument when
-- called from a non-main coroutine and that a non main coroutine is
-- yieldable.
do
    local function cof()
        print("yieldable", coroutine.isyieldable())
        return coroutine.running()
    end
    local co = coroutine.create(cof)

    -- isyieldable does not accept a coroutine argument in our impl
    print(coroutine.isyieldable())
    --> =false

    print(coroutine.resume(co))
    --> =yieldable	true
    --> ~^true\tthread:.*\tfalse$
end

-- Test error in coroutine
do
    local function cof()
        error("boo")
    end
    local co = coroutine.create(cof)
    print(coroutine.resume(co))
    --> ~^false\t.* boo$
    print(coroutine.status(co))
    --> =dead
end

-- Check various statuses of coroutines
do
    local main = coroutine.running()
    local function cof(co)
        coroutine.yield(coroutine.status(co))
        return coroutine.status(main)
    end
    co = coroutine.create(cof)

    print("main/main", coroutine.status(main))
    --> =main/main	running

    print("co/main", coroutine.status(co))
    --> =co/main	suspended

    print("co/co", coroutine.resume(co, co))
    --> =co/co	true	running

    print("main/co", coroutine.resume(co))
    --> =main/co	true	normal

    print("co", coroutine.status(co))
    --> =co	dead
end

-- Test coroutine.wrap()
do
    local function cofib()
        local a, b = 0, 1
        while true do
            coroutine.yield(a)
            a, b = b, a+b
        end
    end
    local fib = coroutine.wrap(cofib)
    print(fib(), fib(), fib(), fib(), fib())
    --> =0	1	1	2	3

    print(pcall(coroutine.wrap))
    --> ~false\t.*function expected

    print(pcall(coroutine.wrap, "not a function"))
    --> ~false\t.*function expected

    local werr = coroutine.wrap(function(x, y)
        coroutine.yield(x)
        error(y)
    end)
    print(werr("abc", "def"))
    --> =abc

    print(pcall(werr))
    --> ~false\t.*def

    print(pcall(werr))
    --> ~false\t.*cannot resume dead coroutine
end

-- Test coroutine.close() (5.4)
do
    print(pcall(coroutine.close))
    --> ~false\t.*thread expected, got no value

    print(pcall(coroutine.close, 1))
    --> ~false\t.*thread expected

    print(pcall(coroutine.close, coroutine.running()))
    --> ~false\t.*cannot close a running coroutine

    local co = coroutine.create(function()
        coroutine.yield()
    end)

    coroutine.resume(co)
    print(coroutine.close(co))
    --> =true
    print(coroutine.status(co))
    --> =dead

    -- __close errors during coroutine.close propagate as false + error message
    local co = coroutine.create(function()
        local t = {}
        setmetatable(t, {__close=function() error("ERR") end})
        local x <close> = t
        coroutine.yield()
    end)
    coroutine.resume(co)
    local ok, msg = coroutine.close(co)
    print(ok, type(msg) == "string" and string.find(msg, "ERR") ~= nil)
    --> =false	true

    print(coroutine.status(co))
    --> =dead
end
