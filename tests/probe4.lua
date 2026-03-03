-- probe4.lua

local function test(name, fn)
    print("TEST " .. name .. ": ")
    local ok, err = pcall(fn)
    if ok then
        print("PASS")
    else
        print("FAIL - " .. tostring(err))
    end
end

test("hex float syntax", function()
    local x = load("return 0x1.8p1")()
    assert(x == 3.0, "Expected 3.0 got " .. tostring(x))
end)

test("unicode string escape", function()
    local x = load([[return "\u{20AC}"]])()
    assert(x == "€", "Expected € got " .. tostring(x))
end)

test("nested block local shadow", function()
    local x = load([[
        local a = 1
        do
            local a = 2
        end
        return a
    ]])()
    assert(x == 1, "Expected 1 got " .. tostring(x))
end)

test("long brackets string", function()
    local s = [==[
        nested [=[ string ]=]
    ]==]
    assert(string.match(s, "nested"), "Should contain nested")
end)

test("bitwise operators precedence", function()
    local x = load("return 1 << 2 + 1")()
    assert(x == 8, "Expected 8 got " .. tostring(x))
end)

test("floor division syntax", function()
    local x = load("return 5.5 // 2")()
    assert(x == 2.0, "Expected 2.0 got " .. tostring(x))
end)
