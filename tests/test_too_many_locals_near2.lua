-- Test that "too many local variables" errors report the correct near token,
-- matching Lua 5.5's behavior for each case (the limit is checked in
-- adjustlocalvars, after the whole declaration is parsed).

-- Helper: generate "local v1, v2, ..., vN = 1,1,...,1\n" to fill N locals
local function makeLocals(n)
    local names = {}
    local vals = {}
    for i = 1, n do
        names[#names+1] = "v" .. i
        vals[#vals+1] = "1"
    end
    return "local " .. table.concat(names, ",") .. " = " .. table.concat(vals, ",") .. "\n"
end

-- Bug 1: local function f() end with 200 locals
-- Lua 5.5 says near '(' because the local for f is registered after parsing '('
do
    local code = makeLocals(200) .. "local function f() end"
    local ok, err = load(code)
    assert(not ok, "should fail with too many locals")
    assert(err:find("near '%('"), "Bug1: expected near '(' but got: " .. err)
end

-- Bug 2: local x <const> = 1 with 200 locals.
-- Lua 5.5 inlines a trailing <const> whose initializer is a compile-time
-- constant (RDKCTC): adjustlocalvars(nvars-1) excludes it from the limit and it
-- consumes no register, so a 201st such const does NOT exceed MAXVARS — the
-- chunk compiles cleanly, matching the reference compiler.
do
    local code = makeLocals(200) .. "local x <const> = 1"
    local ok = load(code)
    assert(ok, "Bug2: trailing inlined <const> must not exceed the local limit")
end

-- Bug 2b: a non-inlinable trailing <const> (initializer is not a compile-time
-- constant) is NOT excluded, so the 201st variable overflows; like the plain
-- case the error reports the token following the whole declaration.
do
    local prefix = ""
    for i = 1, 200 do prefix = prefix .. "local v" .. i .. " = 1\n" end
    local code = prefix .. "local x <const> = print\nprint(1)"
    local ok, err = load(code)
    assert(not ok, "should fail with too many locals")
    assert(err:find("near 'print'"), "Bug2b: expected near 'print' but got: " .. err)
end

-- Bug 3: function with 202+ parameters
-- Lua 5.5 says near ')' because the limit is checked after all params are parsed
do
    local params = {}
    for i = 1, 202 do
        params[#params+1] = "p" .. i
    end
    local code = "function f(" .. table.concat(params, ",") .. ") end"
    local ok, err = load(code)
    assert(not ok, "should fail with too many locals")
    assert(err:find("near '%)'"), "Bug3: expected near ')' but got: " .. err)
end

-- Bug 4: local x1,x2,...,x201 with no assignment
-- Lua 5.5 says near '<eof>' (or the next token) because the limit is reported
-- after the declaration, at the post-statement lookahead token
do
    local names = {}
    for i = 1, 201 do
        names[#names+1] = "x" .. i
    end
    local code = "local " .. table.concat(names, ",")
    local ok, err = load(code)
    assert(not ok, "should fail with too many locals")
    -- With no '=', the error should report the token after the declaration, not 'x201'
    assert(not err:find("near 'x%d+"), "Bug4: should not say near variable name, got: " .. err)
    assert(err:find("near <eof>"), "Bug4: expected near <eof> but got: " .. err)
end

print("OK")
