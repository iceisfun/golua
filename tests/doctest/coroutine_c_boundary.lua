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
