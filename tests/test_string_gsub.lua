-- test_string_gsub: string.gsub replacement modes, edge cases, and error handling

-- Basic replacement with captures
do
    local r, n = string.gsub("hello world", "(%w+)", "%1 %1")
    assert(r == "hello hello world world" and n == 2, "gsub capture dup")
end

-- Replacement with limit
do
    local r, n = string.gsub("hello world", "%w+", "%0 %0", 1)
    assert(r == "hello hello world" and n == 1, "gsub limit")
end

-- Swap captures
do
    local r = string.gsub("hello world from Lua", "(%w+)%s*(%w+)", "%2 %1")
    assert(r == "world hello Lua from", "gsub swap")
end

-- Function replacement
do
    local function getenv(v)
        if v == "HOME" then return "/home/roberto"
        elseif v == "USER" then return "roberto"
        end
    end
    local r, n = string.gsub("home = $HOME, user = $USER", "%$(%w+)", getenv)
    assert(r == "home = /home/roberto, user = roberto" and n == 2, "gsub func")
end

-- Table replacement
do
    local t = {name = "lua", version = "5.3"}
    local r, n = string.gsub("$name-$version.tar.gz", "%$(%w+)", t)
    assert(r == "lua-5.3.tar.gz" and n == 2, "gsub table")
end

-- Percent-escape in replacement
do
    local r, n = string.gsub("xyz", "xyz", "%%")
    assert(r == "%" and n == 1, "gsub percent escape")
end

-- Table replacement with false value = keep original
do
    local r, n = string.gsub("z", "z", {z = false})
    assert(r == "z" and n == 1, "gsub table false keeps original")
end

-- Empty-matching pattern
-- Lua 5.4 semantics: "abc" with pattern "b*" gives 3 replacements
do
    local r, n = string.gsub("abc", "b*", "Z")
    assert(r == "ZaZcZ" and n == 3, "gsub empty match: expected 'ZaZcZ' n=3, got '" .. r .. "' n=" .. n)
end

-- Position captures
do
    local r = string.gsub("xyz", "()y()", "%1-%2")
    assert(r == "x2-3z", "gsub position capture: expected 'x2-3z', got: " .. r)
end

-- Replacement string validation
do
    -- % at end of replacement string should error
    local ok, err = pcall(string.gsub, "abc", "a", "%")
    assert(not ok, "expected error for trailing % in replacement, got: " .. tostring(ok))
    assert(err:find("invalid use of"), "wrong error: " .. tostring(err))

    -- %x (invalid capture reference) should error
    local ok2, err2 = pcall(string.gsub, "abc", "a", "%x")
    assert(not ok2, "expected error for %x in replacement, got: " .. tostring(ok2))

    -- negative n should mean 0 replacements
    local result, count = string.gsub("aaa", "a", "b", -1)
    assert(result == "aaa", "expected 'aaa' with n=-1, got '" .. result .. "'")
    assert(count == 0, "expected count=0 with n=-1, got " .. count)

    -- n=0 should mean 0 replacements (baseline)
    local r0, c0 = string.gsub("aaa", "a", "b", 0)
    assert(r0 == "aaa", "expected 'aaa' with n=0, got '" .. r0 .. "'")
    assert(c0 == 0, "expected count=0 with n=0, got " .. c0)
end

-- Error cases
assert(not pcall(string.gsub), "gsub no args")
assert(not pcall(string.gsub, "x", "y"), "gsub needs 3 args")
assert(not pcall(string.gsub, "x", "y", "z", "t"), "gsub bad limit type")
assert(not pcall(string.gsub, "x", "%", "z"), "gsub malformed pattern")
assert(not pcall(string.gsub, "xyz", "(x)", "%2"), "gsub invalid capture index")
assert(not pcall(string.gsub, "xyz", "xyz", false), "gsub boolean repl")
assert(not pcall(string.gsub, "z", "z", {z = true}), "gsub table true value")
assert(not pcall(string.gsub, "xyz", "xyz", function() error("baa") end), "gsub func error")
