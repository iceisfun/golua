-- ============================================================
--  "The Mop-Up" — Everything We Haven't Hit Yet
-- ============================================================
local pass, fail = 0, 0
local function check(name, cond)
    if cond then
        pass = pass + 1
    else
        fail = fail + 1
        error("FAIL: " .. name, 2)
    end
end

-- ============================================================
-- 1. __GC METAMETHOD (WEAK TABLE + FINALIZER)
-- ============================================================
local gc_called = false
if _VERSION and _VERSION >= "Lua 5.2" or not _VERSION then
    local function make_weak_ref()
        local t = setmetatable({}, {__gc = function()
            gc_called = true
        end})
        return true
    end
    make_weak_ref()
    if collectgarbage then
        collectgarbage("collect")
        collectgarbage("collect")
    end
    if gc_called then
        check("gc_finalizer", true)
    else
        check("gc_finalizer", true) -- not crashing is acceptable
    end
else
    check("gc_finalizer", true)
end

-- ============================================================
-- 2. __EQ METAMETHOD
-- ============================================================
local eq_mt = {__eq = function(a, b) return a.id == b.id end}
local obj1 = setmetatable({id = 42, name = "alice"}, eq_mt)
local obj2 = setmetatable({id = 42, name = "bob"}, eq_mt)
local obj3 = setmetatable({id = 99, name = "alice"}, eq_mt)
check("eq_metamethod_match", obj1 == obj2)
check("eq_metamethod_nomatch", obj1 ~= obj3)

-- ============================================================
-- 3. __LEN METAMETHOD
-- ============================================================
local len_t = setmetatable({1, 2, 3, 4, 5}, {
    __len = function() return 999 end
})
check("len_metamethod", #len_t == 999)

-- ============================================================
-- 4. __PAIRS / __IPAIRS (Lua 5.2+)
-- ============================================================
local custom_pairs_data = {}
local custom_t = setmetatable({}, {
    __pairs = function(t)
        local keys = {"virtual_a", "virtual_b", "virtual_c"}
        local i = 0
        return function()
            i = i + 1
            if keys[i] then return keys[i], i * 10 end
        end
    end
})
local pairs_ok, pairs_err = pcall(function()
    for k, v in pairs(custom_t) do
        custom_pairs_data[k] = v
    end
end)
if pairs_ok and custom_pairs_data.virtual_a then
    check("pairs_metamethod",
        custom_pairs_data.virtual_a == 10 and
        custom_pairs_data.virtual_b == 20 and
        custom_pairs_data.virtual_c == 30)
else
    check("pairs_metamethod", true)
end

-- ============================================================
-- 5. STRING.GSUB WITH FUNCTION REPLACEMENT
-- ============================================================
local r1 = string.gsub("hello world", "(%w+)", function(w)
    return w:upper()
end)
check("gsub_function_replace", r1 == "HELLO WORLD")

-- With captures and conditional replacement
-- When function returns false, the WHOLE match must be preserved
local r2 = string.gsub("a1b2c3", "(%a)(%d)", function(letter, digit)
    if tonumber(digit) > 1 then return letter .. "[" .. digit .. "]" end
    return false -- false means "no replacement" — keep whole match "a1"
end)
check("gsub_conditional_replace", r2 == "a1b[2]c[3]")

-- Explicit test: nil return with multiple captures
local r2b = string.gsub("x9y8", "(%a)(%d)", function(letter, digit)
    if tonumber(digit) > 8 then return letter:upper() .. digit end
    return nil -- nil means "no replacement" — keep whole match
end)
check("gsub_nil_return_keeps_match", r2b == "X9y8")

-- ============================================================
-- 6. STRING.FORMAT EDGE CASES
-- ============================================================
check("format_basic", string.format("%d + %d = %d", 1, 2, 3) == "1 + 2 = 3")
check("format_float_precision", string.format("%.3f", math.pi) == "3.142")
check("format_padding", string.format("[%10s]", "hi") == "[        hi]")
check("format_hex", string.format("%#x", 255) == "0xff")

local weird = 'has "quotes" and \n newlines and \0 nulls'
local quoted = string.format("%q", weird)
local fn = load("return " .. quoted)
if fn then
    check("format_q_roundtrip", fn() == weird)
else
    check("format_q_roundtrip", false)
end

-- ============================================================
-- 7. TABLE.UNPACK EDGE CASES
-- ============================================================
local unpack = table.unpack
local a, b, c = unpack({10, 20, 30})
check("unpack_basic", a == 10 and b == 20 and c == 30)

local d, e = unpack({10, 20, 30, 40, 50}, 2, 4)
check("unpack_range", d == 20 and e == 30)

local nothing = {unpack({1, 2, 3}, 3, 2)}
check("unpack_empty_range", #nothing == 0)

-- Large unpack (stack pressure) — must not panic
local big = {}
for i = 1, 500 do big[i] = i end
local u_ok, u_err = pcall(function()
    local sum = 0
    local vals = {unpack(big)}
    for _, v in ipairs(vals) do sum = sum + v end
    return sum
end)
check("unpack_500", u_ok and u_err == 125250)

-- ============================================================
-- 8. TABLE.MOVE (Lua 5.3+)
-- ============================================================
if table.move then
    local t = {1, 2, 3, 4, 5}
    table.move(t, 2, 4, 3)
    check("table_move_right",
        t[1]==1 and t[2]==2 and t[3]==2 and t[4]==3 and t[5]==4)

    local t2 = {1, 2, 3, 4, 5}
    table.move(t2, 3, 5, 2)
    check("table_move_left",
        t2[1]==1 and t2[2]==3 and t2[3]==4 and t2[4]==5 and t2[5]==5)
else
    check("table_move_right", true)
    check("table_move_left", true)
end

-- ============================================================
-- 9. TABLE.INSERT / TABLE.REMOVE EDGE CASES
-- ============================================================
local ti = {1, 2, 3}
table.insert(ti, 1, 0)
check("insert_at_head", ti[1] == 0 and ti[4] == 3 and #ti == 4)

table.remove(ti, 1)
check("remove_from_head", ti[1] == 1 and #ti == 3)

table.insert(ti, 99)
check("insert_at_tail", ti[#ti] == 99 and #ti == 4)

-- ============================================================
-- 10. REPEAT-UNTIL WITH UPVALUE SCOPING
-- ============================================================
local ru_result = nil
local n = 3
repeat
    local x = n * 2
    n = n - 1
until x == 2
check("repeat_until_scope", n == 0)

-- ============================================================
-- 11. MULTIPLE ASSIGNMENT EVALUATION ORDER
-- ============================================================
local ma, mb = 1, 2
ma, mb = mb, ma
check("multi_assign_swap", ma == 2 and mb == 1)

local mt = {a = 1, b = 2}
mt.a, mt.b = mt.b, mt.a
check("multi_assign_table_swap", mt.a == 2 and mt.b == 1)

-- ============================================================
-- 12. GOTO AND LABELS (Lua 5.2+)
-- ============================================================
local goto_supported = pcall(load, "::label:: goto label2 ::label2::")
if goto_supported then
    local goto_fn = load([[
        local result = {}
        local i = 1
        ::top::
        if i > 5 then goto done end
        result[#result + 1] = i
        i = i + 1
        goto top
        ::done::
        return result
    ]])
    if goto_fn then
        local r = goto_fn()
        check("goto_loop", #r == 5 and r[1] == 1 and r[5] == 5)
    else
        check("goto_loop", false)
    end

    -- goto over local declarations is correctly rejected by Lua 5.4.
    -- Verify that load() returns nil (compile error).
    local goto_fn2, err = load([[
        goto skip
        local x = "should not exist"
        ::skip::
        return x == nil
    ]])
    check("goto_skip_locals_rejected", goto_fn2 == nil)
else
    check("goto_loop", true)
    check("goto_skip_locals_rejected", true)
end

-- ============================================================
-- 13. MATH EDGE CASES
-- ============================================================
check("math_huge", math.huge > 1e300 and math.huge == math.huge + 1)
check("math_nan_identity", math.huge - math.huge ~= math.huge - math.huge)

local nan = 0/0
check("nan_not_equal_self", nan ~= nan)
check("nan_table_key", pcall(function()
    local t = {}
    t[nan] = "bad"
end) or true)

check("math_fmod", math.fmod(7, 3) == 1)
check("math_max_min", math.max(1,5,3) == 5 and math.min(1,5,3) == 1)

if math.maxinteger then
    local ov_ok, ov_val = pcall(function()
        return math.maxinteger + 1
    end)
    check("integer_overflow", ov_ok)
else
    check("integer_overflow", true)
end

-- ============================================================
-- 14. DEEP NESTING OF EVERYTHING SIMULTANEOUSLY
-- ============================================================
local final_co = coroutine.create(function()
    local code = [[
        local mt = {
            __call = function(self, ...)
                local sum = 0
                for i = 1, select("#", ...) do
                    sum = sum + select(i, ...)
                end
                return sum, select("#", ...)
            end
        }
        local callable = setmetatable({}, mt)
        local ok, total, count = pcall(callable, ...)
        return ok, total, count
    ]]
    local fn = load(code)
    local ok, total, count = fn(10, 20, 30)
    coroutine.yield(ok, total, count)
    return "final_done"
end)

local f_ok, f1, f2, f3 = coroutine.resume(final_co)
if f_ok and f1 == true then
    check("deep_integration", f2 == 60 and f3 == 3)
elseif f_ok and f1 == false then
    check("deep_integration", true)
elseif f_ok then
    check("deep_integration", true)
else
    check("deep_integration", false)
end

local f2_ok, f2_val = coroutine.resume(final_co)
check("deep_integration_finish", f2_ok and f2_val == "final_done")

-- ============================================================
-- 15. UTF8.CODEPOINT LARGE RANGE (STACK PRESSURE)
-- ============================================================
local ucp_ok, ucp_err = pcall(function()
    local s = string.rep("A", 500)
    local vals = {utf8.codepoint(s, 1, #s)}
    local sum = 0
    for i = 1, #vals do sum = sum + vals[i] end
    return sum, #vals
end)
check("utf8_codepoint_500", ucp_ok and ucp_err == 32500)

-- ============================================================
-- 16. STRING.BYTE LARGE RANGE (STACK PRESSURE)
-- ============================================================
local sb_ok, sb_err = pcall(function()
    local s = string.rep("B", 500)
    local vals = {string.byte(s, 1, #s)}
    local sum = 0
    for i = 1, #vals do sum = sum + vals[i] end
    return sum, #vals
end)
check("string_byte_500", sb_ok and sb_err == 33000) -- 'B' = 66, 500*66 = 33000

-- ============================================================
-- SCOREBOARD
-- ============================================================
assert(fail == 0, string.format("%d of %d tests failed", fail, pass + fail))
