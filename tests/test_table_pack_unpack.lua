-- Tests for table.pack and table.unpack (Lua 5.2+ semantics).
-- Validates nil-handling, argument count preservation, and round-trip correctness.

local function assert_eq(a, b, msg)
    if a ~= b then
        error(msg or (tostring(a) .. " ~= " .. tostring(b)), 2)
    end
end

local function assert_true(v, msg)
    if not v then
        error(msg or "assertion failed", 2)
    end
end

-- Test 1: Basic table.pack
do
    local t = table.pack(1, 2, 3)
    assert_eq(t.n, 3, "pack basic: n")
    assert_eq(t[1], 1, "pack basic: [1]")
    assert_eq(t[2], 2, "pack basic: [2]")
    assert_eq(t[3], 3, "pack basic: [3]")
end

-- Test 2: table.pack with nil in middle
do
    local t = table.pack(1, nil, 3)
    assert_eq(t.n, 3, "pack nil middle: n")
    assert_eq(t[1], 1, "pack nil middle: [1]")
    assert_eq(t[2], nil, "pack nil middle: [2]")
    assert_eq(t[3], 3, "pack nil middle: [3]")
end

-- Test 3: table.pack with trailing nils
do
    local t = table.pack(1, 2, nil, nil)
    assert_eq(t.n, 4, "pack trailing nils: n")
    assert_eq(t[1], 1, "pack trailing nils: [1]")
    assert_eq(t[2], 2, "pack trailing nils: [2]")
    assert_eq(t[3], nil, "pack trailing nils: [3]")
    assert_eq(t[4], nil, "pack trailing nils: [4]")
end

-- Test 4: table.pack with all nil values
do
    local t = table.pack(nil, nil, nil)
    assert_eq(t.n, 3, "pack all nil: n")
    assert_eq(t[1], nil, "pack all nil: [1]")
    assert_eq(t[2], nil, "pack all nil: [2]")
    assert_eq(t[3], nil, "pack all nil: [3]")
end

-- Test 5: table.pack with zero arguments
do
    local t = table.pack()
    assert_eq(t.n, 0, "pack zero args: n")
    assert_true(next(t) == nil or (next(t) == "n" and next(t, "n") == nil),
        "pack zero args: table should only contain n")
end

-- Test 6: Basic table.unpack
do
    local a, b, c = table.unpack({1, 2, 3})
    assert_eq(a, 1, "unpack basic: a")
    assert_eq(b, 2, "unpack basic: b")
    assert_eq(c, 3, "unpack basic: c")
end

-- Test 7: table.unpack with explicit range
do
    local t = {10, 20, 30, 40}
    local a, b = table.unpack(t, 2, 3)
    assert_eq(a, 20, "unpack range: a")
    assert_eq(b, 30, "unpack range: b")
end

-- Test 8: table.unpack respecting n
do
    local t = table.pack(1, nil, 3)
    local a, b, c = table.unpack(t, 1, t.n)
    assert_eq(a, 1, "unpack respecting n: a")
    assert_eq(b, nil, "unpack respecting n: b")
    assert_eq(c, 3, "unpack respecting n: c")
end

-- Test 9: Round-trip pack/unpack
do
    local t1 = table.pack(1, nil, 3, nil, 5)
    local t2 = table.pack(table.unpack(t1, 1, t1.n))
    assert_eq(t2.n, t1.n, "round-trip: n mismatch")
    for i = 1, t1.n do
        assert_eq(t1[i], t2[i], "round-trip: mismatch at index " .. i)
    end
end

-- Test 10: Variadic forwarding pattern
do
    local function f(...)
        local args = table.pack(...)
        return table.unpack(args, 1, args.n)
    end

    local a, b, c = f(1, nil, 3)
    assert_eq(a, 1, "variadic forward: a")
    assert_eq(b, nil, "variadic forward: b")
    assert_eq(c, 3, "variadic forward: c")
end

-- Test 11: Large range unpack (range exceeds table length)
do
    local t = {1, 2}
    local a, b, c, d = table.unpack(t, 1, 4)
    assert_eq(a, 1, "large range: a")
    assert_eq(b, 2, "large range: b")
    assert_eq(c, nil, "large range: c")
    assert_eq(d, nil, "large range: d")
end
