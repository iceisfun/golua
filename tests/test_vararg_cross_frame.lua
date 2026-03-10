-- Test that vararg values survive across calls to other vararg functions.
-- Previously, varargs were stored on the shared stack at a fixed offset
-- from the frame base, causing callee vararg frames to overwrite caller's
-- vararg values.

local function eat(...) end

-- Basic: 3 args should survive after calling another vararg function
do
    local function test(...)
        eat(...)
        local a, b, c = ...
        assert(a == "a" and b == "b" and c == "c",
            "3 args corrupted: " .. tostring(a) .. "," .. tostring(b) .. "," .. tostring(c))
    end
    test("a", "b", "c")
end

-- 5 args
do
    local function test(...)
        eat(...)
        local a, b, c, d, e = ...
        assert(a == 1 and b == 2 and c == 3 and d == 4 and e == 5,
            "5 args corrupted")
    end
    test(1, 2, 3, 4, 5)
end

-- Nested vararg calls
do
    local function inner(...)
        local function deeper(...) end
        deeper(...)
        return ...
    end
    local function outer(...)
        local r1, r2, r3 = inner(...)
        assert(r1 == "x" and r2 == "y" and r3 == "z",
            "nested vararg corrupted: " .. tostring(r1) .. "," .. tostring(r2) .. "," .. tostring(r3))
    end
    outer("x", "y", "z")
end

-- select('#', ...) should be correct after vararg call
do
    local function test(...)
        eat(...)
        assert(select('#', ...) == 4, "vararg count changed")
        local a, b, c, d = ...
        assert(a == "w" and b == "x" and c == "y" and d == "z")
    end
    test("w", "x", "y", "z")
end

-- Real-world pattern: helper function then use varargs
do
    local function helper(...) return select('#', ...) end
    local function process(...)
        local count = helper(...)
        local a, b, c = ...
        assert(count == 3 and a == "p" and b == "q" and c == "r",
            "helper+vararg corrupted")
    end
    process("p", "q", "r")
end

print("PASS")
