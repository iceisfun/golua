-- Test: register-limit boundary matches reference Lua exactly (off-by-one guard).
--
-- Reference Lua's luaK_checkstack errors when the new stack top is >= MAXREGS
-- (255), so the highest usable register count is 254. golua previously used a
-- `>` comparison instead of `>=`, accepting one register past the reference
-- limit (e.g. a 255-value return compiled here but errored under reference Lua).

local function repeated(n, sep)
    local t = {}
    for i = 1, n do t[i] = "1" end
    return table.concat(t, sep)
end

-- A function returning N constants reserves N registers. At the boundary:
--   N = 254  -> compiles OK
--   N = 255  -> "function or expression needs too many registers"
local okSrc = "local function f() return " .. repeated(254, ", ") .. " end return f"
local f, err = load(okSrc)
assert(f, "254-value return should compile, got error: " .. tostring(err))

local badSrc = "local function f() return " .. repeated(255, ", ") .. " end"
f, err = load(badSrc)
assert(not f, "255-value return should fail to compile (reference rejects it)")
assert(string.find(err, "too many registers", 1, true),
       "expected 'too many registers' for 255-value return, got: " .. tostring(err))

-- Function-call arguments hit the same boundary one slot lower (the called
-- function occupies a register): reference rejects 253 args, accepts 252.
local okArgs = "local function f() end f(" .. repeated(252, ", ") .. ") return 1"
f, err = load(okArgs)
assert(f, "252-arg call should compile, got error: " .. tostring(err))

local badArgs = "local function f() end f(" .. repeated(253, ", ") .. ")"
f, err = load(badArgs)
assert(not f, "253-arg call should fail to compile (reference rejects it)")
assert(string.find(err, "too many registers", 1, true),
       "expected 'too many registers' for 253-arg call, got: " .. tostring(err))

print("OK")
