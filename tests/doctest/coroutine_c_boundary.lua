-- Coroutine yieldability at C-call boundaries.
-- Lua 5.4 behavior:
-- - pcall(...) over a Lua function remains yieldable
-- - yielding inside non-yieldable native callback contexts must fail

do
    local co = coroutine.create(function()
        local ok = pcall(function()
            coroutine.yield("Y")
        end)
        return ok
    end)

    print(coroutine.resume(co))
    --> =true	Y
    print(coroutine.resume(co))
    --> =true	true
end

do
    local co = coroutine.create(function()
        return pcall(table.sort, {3, 2, 1}, function(a, b)
            return coroutine.yield(a, b)
        end)
    end)

    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

do
    local co = coroutine.create(function()
        return pcall(string.gsub, "abc", ".", function(ch)
            return coroutine.yield(ch)
        end)
    end)

    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

do
    local co = coroutine.create(function()
        return string.gsub("ab", ".", function()
            return tostring(coroutine.isyieldable())
        end)
    end)

    print(coroutine.resume(co))
    --> =true	falsefalse	2
end

-- A metamethod triggered *implicitly* from inside a C-level library function
-- (not via an explicit Lua callback) must also be non-yieldable. Reference Lua
-- raises "attempt to yield across a C-call boundary" for each of these.

-- table.sort with the default '<' comparator -> __lt
do
    local mt = {__lt = function(a, b) coroutine.yield(); return rawget(a, "v") < rawget(b, "v") end}
    local co = coroutine.create(function()
        local t = {setmetatable({v = 3}, mt), setmetatable({v = 1}, mt)}
        return pcall(table.sort, t)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- table.insert shifting elements -> __newindex
do
    local mt = {__newindex = function(t, k, val) coroutine.yield(); rawset(t, k, val) end}
    local co = coroutine.create(function()
        local t = setmetatable({}, mt)
        rawset(t, 1, "a")
        return pcall(table.insert, t, 1, "z")
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- table.unpack reading an absent index -> __index
do
    local mt = {__index = function(_, k) coroutine.yield(); return k end}
    local co = coroutine.create(function()
        local t = setmetatable({}, mt)
        return pcall(table.unpack, t, 1, 2)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- table.move writing an absent index -> __newindex
do
    local mt = {__newindex = function(t, k, val) coroutine.yield(); rawset(t, k, val) end}
    local co = coroutine.create(function()
        local t = setmetatable({}, mt)
        rawset(t, 1, "a")
        rawset(t, 2, "b")
        return pcall(table.move, t, 1, 2, 3)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- table.concat reading an absent index -> __index
do
    local mt = {__index = function(_, k) coroutine.yield(); return tostring(k) end}
    local co = coroutine.create(function()
        local t = setmetatable({}, mt)
        return pcall(table.concat, t, ",", 1, 2)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- ipairs iterator reading via __index
do
    local mt = {__index = function(_, k) coroutine.yield(); if k <= 2 then return k end end}
    local co = coroutine.create(function()
        local t = setmetatable({}, mt)
        return pcall(function()
            for _ in ipairs(t) do end
            return true
        end)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end

-- string.gsub with a table replacement reading an absent key -> __index
do
    local mt = {__index = function(_, _) coroutine.yield(); return "Z" end}
    local co = coroutine.create(function()
        local repl = setmetatable({}, mt)
        return pcall(string.gsub, "a", "%a", repl)
    end)
    print(coroutine.resume(co))
    --> ~^true\tfalse\t.*yield across a C-call boundary$
end
