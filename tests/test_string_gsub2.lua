-- test_string_gsub2: string.gsub replacement modes and edge cases

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

-- Error cases
assert(not pcall(string.gsub), "gsub no args")
