-- test_string_gmatch: string.gmatch iteration and captures

-- Basic digit chunk iteration
do
    local matches = {}
    for token in string.gmatch("abc123def456", "%d+") do
        matches[#matches + 1] = token
    end
    assert(#matches == 2, string.format("expected 2 digit chunks, got %d", #matches))
    assert(matches[1] == "123" and matches[2] == "456", string.format("unexpected tokens: %s", table.concat(matches, ",")))
end

-- Key-value captures
do
    local assignments = {}
    for name, value in string.gmatch("foo=1 bar=20 baz=300", "(%a+)=(%d+)") do
        assignments[name] = value
    end
    assert(assignments.foo == "1", "missing capture 'foo'")
    assert(assignments.bar == "20", "missing capture 'bar'")
    assert(assignments.baz == "300", "missing capture 'baz'")
end

-- Word iteration
do
    local t = {}
    for w in string.gmatch("hello world from Lua", "%a+") do
        t[#t + 1] = w
    end
    assert(#t == 4, "gmatch word count")
    assert(t[1] == "hello" and t[2] == "world" and t[3] == "from" and t[4] == "Lua",
        "gmatch words")
end

-- Key-value captures (variant)
do
    local t = {}
    for k, v in string.gmatch("from=world, to=Lua", "(%w+)=(%w+)") do
        t[k] = v
    end
    assert(t.from == "world" and t.to == "Lua", "gmatch kv captures")
end

-- Empty pattern matches
-- Lua 5.4 semantics: lastMatch check prevents duplicate empty match at same position
-- gmatch("abc", "b*") -> "", "b", "" (3 results, not 4)
do
    local t = {}
    for w in string.gmatch("abc", "b*") do
        t[#t + 1] = w
    end
    assert(#t == 3, "gmatch empty match count: " .. #t)
    assert(t[1] == "" and t[2] == "b" and t[3] == "",
        "gmatch empty match values")
end

-- Error cases
assert(not pcall(string.gmatch), "gmatch no args")
assert(not pcall(string.gmatch, "x"), "gmatch 1 arg")

-- Optional init argument (Lua 5.4)
do
    local t = {}
    for w in string.gmatch("hello world from Lua", "%a+", 8) do
        t[#t + 1] = w
    end
    assert(t[1] == "orld", "gmatch init should start at position 8, got: " .. tostring(t[1]))
    assert(#t == 3, "expected 3 words after position 8")
end

-- Negative init
do
    local t2 = {}
    for w in string.gmatch("hello world from Lua", "%a+", -7) do
        t2[#t2 + 1] = w
    end
    assert(t2[1] == "rom", "gmatch negative init")
end
