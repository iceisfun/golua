-- probe.lua

local function test(name, fn)
    print("TEST " .. name .. ": ")
    local ok, err = pcall(fn)
    if ok then
        print("PASS")
    else
        print("FAIL - " .. tostring(err))
    end
end

test("len operator on holes", function()
    local t = {1, 2, nil, 4}
    local l = #t
    assert(l == 2 or l == 4, "Length should be 2 or 4, got " .. tostring(l))
end)

test("bitwise string coercion", function()
    local res = "10" | 2
    assert(res == 10 | 2)
end)

test("close metamethod error", function()
    local closed = false
    local function inner()
        local x <close> = setmetatable({}, {
            __close = function(self)
                closed = true
                error("error in close")
            end
        })
        error("error in body")
    end
    local ok, err = pcall(inner)
    assert(not ok, "should fail")
    assert(closed, "should have been closed")
    -- In Lua 5.4, if body errors and close errors, the close error is reported if it's the first one, 
    -- but actually body error happens first, then close is called, if close errors it might wrap or override.
end)

test("coroutine yield across pcall", function()
    local co = coroutine.create(function()
        pcall(function()
            coroutine.yield("yielded")
        end)
    end)
    local ok, res = coroutine.resume(co)
    assert(ok and res == "yielded", "Expected to yield across pcall")
end)

test("table iteration modification", function()
    local t = {a=1, b=2, c=3}
    local count = 0
    for k, v in pairs(t) do
        count = count + 1
        t.d = 4 -- inserting new key during iteration
    end
    -- behavior is undefined in standard lua, but let's see if it crashes
end)

test("metatable __index loop", function()
    local a = {}
    local b = setmetatable({}, {__index = a})
    setmetatable(a, {__index = b})
    local ok, err = pcall(function() return b.missing end)
    assert(not ok, "should error on loop")
end)

test("math.random bounds", function()
    local r = math.random(0)
    -- math.random(0) is valid in Lua 5.4: generates full integer
end)

test("goto into scope", function()
    local fn = load([[
        goto l
        do
            local x = 1
            ::l::
            print(x)
        end
    ]])
    assert(not fn, "Should not compile")
end)
