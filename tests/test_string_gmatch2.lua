-- test_string_gmatch2: string.gmatch iteration and captures

-- Basic word iteration
do
    local t = {}
    for w in string.gmatch("hello world from Lua", "%a+") do
        t[#t + 1] = w
    end
    assert(#t == 4, "gmatch word count")
    assert(t[1] == "hello" and t[2] == "world" and t[3] == "from" and t[4] == "Lua",
        "gmatch words")
end

-- Key-value captures
do
    local t = {}
    for k, v in string.gmatch("from=world, to=Lua", "(%w+)=(%w+)") do
        t[k] = v
    end
    assert(t.from == "world" and t.to == "Lua", "gmatch kv captures")
end

-- Empty pattern matches (b* matches empty between each char)
-- Lua 5.4 semantics: lastMatch check prevents duplicate empty match at same position
-- gmatch("abc", "b*") → "", "b", "" (3 results, not 4)
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
