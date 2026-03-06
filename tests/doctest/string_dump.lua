-- string.dump / load round-trip
-- Serialize a Lua function to binary bytecode and reload it.

-- Basic round-trip
do
    local function square(x) return x * x end
    local dumped = string.dump(square)
    print(type(dumped))
    --> =string
    local loaded = load(dumped)
    print(loaded(7))
    --> =49
end

-- Stripped dump (no debug info)
do
    local function add(a, b) return a + b end
    local full = string.dump(add)
    local stripped = string.dump(add, true)
    print(#stripped <= #full)
    --> =true
    local f = load(stripped)
    print(f(10, 20))
    --> =30
end

-- Multiple return values survive dump/load
do
    local function multi() return 1, "two", true end
    local f = load(string.dump(multi))
    print(f())
    --> =1	two	true
end

-- Nested closures
do
    local function make_adder(n)
        return function(x) return x + n end
    end
    local f = load(string.dump(make_adder))
    local add5 = f(5)
    print(add5(10))
    --> =15
end

-- Environment override: loaded function uses custom _ENV
do
    local function get_x() return x end
    local env = setmetatable({x = 42}, {__index = _G})
    local f = load(string.dump(get_x), nil, nil, env)
    print(f())
    --> =42
end

-- Mode checks
do
    local d = string.dump(function() return 1 end)
    -- mode "b" accepts binary
    print(load(d, nil, "b")())
    --> =1
    -- mode "t" rejects binary
    local f, err = load(d, nil, "t")
    print(f == nil)
    --> =true
end

-- Error: dumping a native function
do
    local ok, err = pcall(string.dump, print)
    print(ok)
    --> =false
end

-- Error: non-function argument
do
    local ok, err = pcall(string.dump, 42)
    print(ok)
    --> =false
end
