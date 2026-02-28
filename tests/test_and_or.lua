-- test_and_or: and/or ternary pattern and multi-call register allocation

-- and/or ternary via function
do
    local function yn(v) return v and "Y" or "N" end

    -- Multiple calls as format args
    local a, b, c = true, true, true
    assert(string.format("A:%s B:%s C:%s", yn(a), yn(b), yn(c)) == "A:Y B:Y C:Y",
        "ternary in format multiple calls")

    -- Mixed values
    local r = string.format("A:%s B:%s C:%s", yn(true), yn(false), yn(true))
    assert(r == "A:Y B:N C:Y", "ternary mixed values")

    -- Four calls
    local r2 = string.format("A:%s B:%s C:%s D:%s", yn(true), yn(true), yn(true), yn(true))
    assert(r2 == "A:Y B:Y C:Y D:Y", "ternary four calls")

    -- Return value type (string, not boolean)
    local val = yn(true)
    assert(type(val) == "string", "yn should return string, got " .. type(val))
    assert(val == "Y", "yn(true) should be 'Y', got " .. val)

    -- Chain preserves type
    local r1 = yn(true)
    local r2a = yn(true)
    local r3 = yn(true)
    assert(type(r1) == "string" and type(r2a) == "string" and type(r3) == "string",
        "and/or chain should preserve string type")
    assert(r1 == "Y" and r2a == "Y" and r3 == "Y",
        "and/or chain should all be 'Y'")
end

-- Inline and/or ternary (not wrapped in function)
do
    local a, b, c = true, true, true
    assert(string.format("A:%s B:%s C:%s",
        a and "Y" or "N",
        b and "Y" or "N",
        c and "Y" or "N") == "A:Y B:Y C:Y",
        "inline ternary")
end

-- Method-style call with and/or ternary
do
    local function has_state(name) return true end
    local bc = has_state("Battle Command")
    local bo = has_state("Battle Orders")
    local sh = has_state("Shout")
    local function yn(v) return v and "Y" or "N" end
    assert(string.format("BC:%s BO:%s Shout:%s", yn(bc), yn(bo), yn(sh)) == "BC:Y BO:Y Shout:Y",
        "method call args")
end

-- Multi-call return freeReg gap
do
    local function double(x) return x * 2 end
    local function test()
        return double(10), double(20), double(30)
    end
    local a, b, c = test()
    assert(a == 20, "multi-call return a")
    assert(b == 40, "multi-call return b")
    assert(c == 60, "multi-call return c")
end

-- Multi-arg functions (more register pressure)
do
    local function add(a, b) return a + b end
    local r1, r2, r3 = add(1, 2), add(3, 4), add(5, 6)
    assert(r1 == 3 and r2 == 7 and r3 == 11, "multi-arg funcs")
end

-- Nested calls as last arg
do
    local function double(x) return x * 2 end
    local function inc(x) return x + 1 end
    assert(string.format("%d %d %d", double(1), double(2), inc(double(3))) == "2 4 7",
        "nested calls as last arg")
end

-- String concat in function args
do
    local function wrap(s) return "[" .. s .. "]" end
    local a, b, c = wrap("a"), wrap("b"), wrap("c")
    assert(a == "[a]" and b == "[b]" and c == "[c]", "string concat in func args")
end
