-- Bug: _ENV (globals) get corrupted when many locals accumulate in a single
-- function scope combined with high temporary register usage from function
-- calls with multiple arguments and complex expressions.
--
-- Symptoms: globals like error, print, math resolve to wrong values (e.g.
-- print becomes setmetatable) or become nil.
--
-- This file tests various scenarios of high local + register pressure
-- to ensure globals remain accessible via direct _ENV access.

-- Save references early so we can report failures even if globals are corrupted.
local saved_assert = assert
local saved_type = type
local saved_tostring = tostring

-- Check globals via DIRECT global access (not saved references).
-- This is the key: the bug corrupts the _ENV upvalue, so we must
-- read globals through _ENV to detect it.
local function check_globals(label)
    saved_assert(type(error) == "function",
        label .. ": error should be function, got " .. saved_type(error))
    saved_assert(type(print) == "function",
        label .. ": print should be function, got " .. saved_type(print))
    saved_assert(type(math) == "table",
        label .. ": math should be table, got " .. saved_type(math))
    saved_assert(type(string) == "table",
        label .. ": string should be table, got " .. saved_type(string))
    saved_assert(type(table) == "table",
        label .. ": table should be table, got " .. saved_type(table))
    saved_assert(type(assert) == "function",
        label .. ": assert should be function, got " .. saved_type(assert))
    saved_assert(type(pcall) == "function",
        label .. ": pcall should be function, got " .. saved_type(pcall))
    saved_assert(type(tostring) == "function",
        label .. ": tostring should be function, got " .. saved_type(tostring))
    -- Behavioral: verify math.pi is accessible and correct
    saved_assert(type(math.pi) == "number",
        label .. ": math.pi should be number")
    saved_assert(math.pi > 3.14 and math.pi < 3.15,
        label .. ": math.pi value wrong")
    -- Behavioral: verify tostring works
    saved_assert(tostring(42) == "42",
        label .. ": tostring(42) broken")
    -- Behavioral: verify string.len works
    saved_assert(string.len("hello") == 5,
        label .. ": string.len broken")
    -- Behavioral: verify table.concat works
    saved_assert(table.concat({"a","b"}, ",") == "a,b",
        label .. ": table.concat broken")
end

-- Helper that takes 4 args (pushes 5 temp regs per call)
local function T(name, ok, got, expected)
    if not ok then
        saved_assert(false, name .. ": got " .. saved_tostring(got)
            .. ", expected " .. saved_tostring(expected))
    end
end

-- ============================================================
-- Test 1: Globals after many locals in do-end blocks
-- ============================================================
do
    local v01,v02,v03,v04,v05,v06,v07,v08,v09,v10=1,2,3,4,5,6,7,8,9,10
    local v11,v12,v13,v14,v15,v16,v17,v18,v19,v20=1,2,3,4,5,6,7,8,9,10
    local v21,v22,v23,v24,v25,v26,v27,v28,v29,v30=1,2,3,4,5,6,7,8,9,10
    local v31,v32,v33,v34,v35,v36,v37,v38,v39,v40=1,2,3,4,5,6,7,8,9,10
    local v41,v42,v43,v44,v45,v46,v47,v48,v49,v50=1,2,3,4,5,6,7,8,9,10
    T("t1_a", true, 1, 1)
    T("t1_b", true, 1, 1)
    T("t1_c", true, 1, 1)
    T("t1_d", true, 1, 1)
    T("t1_e", true, 1, 1)
    check_globals("block_50_locals")
end

-- ============================================================
-- Test 2: Globals after goto in a block
-- ============================================================
do
    local v01,v02,v03,v04,v05,v06,v07,v08,v09,v10=1,2,3,4,5,6,7,8,9,10
    local v11,v12,v13,v14,v15,v16,v17,v18,v19,v20=1,2,3,4,5,6,7,8,9,10
    local v21,v22,v23,v24,v25,v26,v27,v28,v29,v30=1,2,3,4,5,6,7,8,9,10
    local val = 0
    goto skip1
    val = 1
    ::skip1::
    val = 2
    saved_assert(val == 2, "goto basic: " .. val)
    T("t2_a", true, 1, 1)
    T("t2_b", true, 1, 1)
    T("t2_c", true, 1, 1)
    check_globals("block_goto_30_locals")
end

-- ============================================================
-- Test 3: Coroutines + locals + function calls in block
-- ============================================================
do
    local co1 = coroutine.create(function(x) coroutine.yield(x+1); return x+2 end)
    local co2 = coroutine.create(function(x) coroutine.yield(x*2); return x*3 end)
    local ok1, v1 = coroutine.resume(co1, 10)
    local ok2, v2 = coroutine.resume(co2, 10)
    T("co_v1", v1 == 11, v1, 11)
    T("co_v2", v2 == 20, v2, 20)
    local ok3, v3 = coroutine.resume(co1)
    local ok4, v4 = coroutine.resume(co2)
    T("co_v3", v3 == 12, v3, 12)
    T("co_v4", v4 == 30, v4, 30)
    check_globals("block_coroutines")
end

-- ============================================================
-- Test 4: Complex expressions with many temporaries
-- ============================================================
do
    local a, b, c = 10, 20, 30
    T("expr1", a + b + c == 60, a + b + c, 60)
    T("expr2", math.type(3 + 4) == "integer", math.type(3 + 4), "integer")
    T("expr3", math.type(3 + 4.0) == "float", math.type(3 + 4.0), "float")
    T("expr4", math.type(3 * 4) == "integer", math.type(3 * 4), "integer")
    T("expr5", math.type(7 // 2) == "integer", math.type(7 // 2), "integer")
    T("expr6", math.type(7 / 2) == "float", math.type(7 / 2), "float")
    T("expr7", string.format("%d+%d=%d", a, b, a+b) == "10+20=30",
      string.format("%d+%d=%d", a, b, a+b), "10+20=30")
    T("expr8", math.max(a, b, c) == 30, math.max(a, b, c), 30)
    check_globals("block_complex_exprs")
end

-- ============================================================
-- Test 5: Nested do-end blocks
-- ============================================================
do
    local v01,v02,v03,v04,v05 = 1,2,3,4,5
    do
        local v06,v07,v08,v09,v10 = 6,7,8,9,10
        do
            local v11,v12,v13,v14,v15 = 11,12,13,14,15
            do
                local v16,v17,v18,v19,v20 = 16,17,18,19,20
                check_globals("nested_inner")
            end
            check_globals("nested_mid")
        end
        check_globals("nested_outer")
    end
    check_globals("nested_done")
end

-- ============================================================
-- Test 6: Metamethods + locals + T calls (realistic mix)
-- ============================================================
do
    local mt_lt = {__lt = function(a,b) return a.v < b.v end}
    local a1 = setmetatable({v=1}, mt_lt)
    local b1 = setmetatable({v=2}, mt_lt)
    T("meta_lt", a1 < b1, a1 < b1, true)

    local mt_ar = {
        __add = function(a,b) return (type(a)=="table" and a.v or a) + (type(b)=="table" and b.v or b) end,
    }
    local c1 = setmetatable({v=10}, mt_ar)
    T("meta_add", c1 + 5 == 15, c1 + 5, 15)

    local s1, e1, cap1 = string.find("hello123", "(%d+)")
    T("find", cap1 == "123", cap1, "123")
    local words = {}
    for w in string.gmatch("one two three", "%a+") do words[#words+1] = w end
    T("gmatch", #words == 3, #words, 3)
    local t1 = {3, 1, 4, 1, 5}
    table.sort(t1)
    T("sort", t1[1] == 1, t1[1], 1)
    check_globals("block_realistic_mix")
end

-- ============================================================
-- Test 7: Multi-return table constructors + T calls
-- ============================================================
do
    local function multi() return 1, 2, 3 end
    local mt = {multi()}
    T("multi_tbl", mt[1]==1 and mt[2]==2 and mt[3]==3, mt[1], 1)
    local mt2 = {multi(), "x"}
    T("multi_adj", mt2[1]==1 and mt2[2]=="x", mt2[2], "x")
    check_globals("block_multi_return")
end

-- ============================================================
-- Tests 8-19: Top-level local accumulation
-- (no do-end block protection — this is the pattern that triggers the bug)
-- ============================================================

-- Accumulate many top-level locals
local A = setmetatable({v=1}, {__lt = function(a,b) return a.v < b.v end})
local B = setmetatable({v=2}, {__lt = function(a,b) return a.v < b.v end})
T("tl_lt", A < B, A < B, true)
check_globals("tl_after_2_meta_locals")

local C = setmetatable({v=1}, {__le = function(a,b) return a.v <= b.v end})
local D = setmetatable({v=1}, {__le = function(a,b) return a.v <= b.v end})
T("tl_le", C <= D, C <= D, true)

local mt1 = {__lt = function(a,b) return a.v < b.v end}
local mt2 = {__lt = function(a,b) return a.v < b.v end}
local E = setmetatable({v=1}, mt1)
local F = setmetatable({v=2}, mt2)
T("tl_lt_diff_mt", E < F, E < F, true)

local lenT = setmetatable({}, {__len = function() return 42 end})
T("tl_len", #lenT == 42, #lenT, 42)

local concT = setmetatable({}, {__concat = function(a,b) return "cat" end})
T("tl_concat", concT .. "x" == "cat", concT .. "x", "cat")

local unmT = setmetatable({}, {__unm = function(a) return 99 end})
T("tl_unm", -unmT == 99, -unmT, 99)

local idxT = setmetatable({}, {__index = function(t, k) return k .. "!" end})
T("tl_index", idxT.hello == "hello!", idxT.hello, "hello!")

local niLog = {}
local niT = setmetatable({}, {__newindex = function(t, k, v) niLog[#niLog+1] = k end})
niT.x = 1
T("tl_newindex", niLog[1] == "x", niLog[1], "x")

local callT = setmetatable({}, {__call = function(t, a, b) return a + b end})
T("tl_call", callT(3, 4) == 7, callT(3, 4), 7)
check_globals("tl_after_metamethod_locals")

-- Arithmetic metamethods
local mt_arith = {
    __add = function(a,b) return (type(a)=="table" and a.v or a) + (type(b)=="table" and b.v or b) end,
    __sub = function(a,b) return (type(a)=="table" and a.v or a) - (type(b)=="table" and b.v or b) end,
    __mul = function(a,b) return (type(a)=="table" and a.v or a) * (type(b)=="table" and b.v or b) end,
    __div = function(a,b) return (type(a)=="table" and a.v or a) / (type(b)=="table" and b.v or b) end,
}
local arT = setmetatable({v=10}, mt_arith)
T("tl_add", arT + 5 == 15, arT + 5, 15)
T("tl_sub", arT - 3 == 7, arT - 3, 7)
T("tl_mul", arT * 2 == 20, arT * 2, 20)
T("tl_div", arT / 4 == 2.5, arT / 4, 2.5)
check_globals("tl_after_arith_meta")

-- Bitwise metamethods
local mt_bit = {
    __band = function(a,b) return (type(a)=="table" and a.v or a) & (type(b)=="table" and b.v or b) end,
    __bor = function(a,b) return (type(a)=="table" and a.v or a) | (type(b)=="table" and b.v or b) end,
    __bxor = function(a,b) return (type(a)=="table" and a.v or a) ~ (type(b)=="table" and b.v or b) end,
    __bnot = function(a) return ~a.v end,
    __shl = function(a,b) return (type(a)=="table" and a.v or a) << (type(b)=="table" and b.v or b) end,
    __shr = function(a,b) return (type(a)=="table" and a.v or a) >> (type(b)=="table" and b.v or b) end,
}
local btT = setmetatable({v=0xFF}, mt_bit)
T("tl_band", btT & 0x0F == 0x0F, btT & 0x0F, 0x0F)
T("tl_bor", btT | 0x100 == 0x1FF, btT | 0x100, 0x1FF)
T("tl_bxor", btT ~ 0xFF == 0, btT ~ 0xFF, 0)
T("tl_bnot", ~btT == ~0xFF, ~btT, ~0xFF)
T("tl_shl", btT << 4 == 0xFF0, btT << 4, 0xFF0)
T("tl_shr", btT >> 4 == 0x0F, btT >> 4, 0x0F)
check_globals("tl_after_bitwise_meta")

-- String operations
local s1, e1, c1 = string.find("hello123", "(%d+)")
T("tl_sfind", c1 == "123", c1, "123")
local words = {}
for w in string.gmatch("one two three", "%a+") do words[#words+1] = w end
T("tl_gmatch", #words == 3 and words[2] == "two", #words, 3)
local result, count = string.gsub("aaa", "a", "b")
T("tl_gsub", result == "bbb" and count == 3, result, "bbb")
local r2 = string.gsub("hello", "(%w+)", function(w) return w:upper() end)
T("tl_gsub_fn", r2 == "HELLO", r2, "HELLO")
local b1, b2, b3 = string.byte("ABC", 1, 3)
T("tl_byte", b1 == 65 and b2 == 66 and b3 == 67, b1, 65)
check_globals("tl_after_string_ops")

-- More string operations (format, rep, reverse, sub, etc.)
T("tl_srep_sep", string.rep("ab", 3, ",") == "ab,ab,ab", string.rep("ab", 3, ","), "ab,ab,ab")
T("tl_srep_0", string.rep("ab", 0) == "", string.rep("ab", 0), "")
T("tl_srev", string.reverse("hello") == "olleh", string.reverse("hello"), "olleh")
T("tl_ssub_neg", string.sub("hello", -3) == "llo", string.sub("hello", -3), "llo")
T("tl_ssub_neg2", string.sub("hello", -4, -2) == "ell", string.sub("hello", -4, -2), "ell")
T("tl_sfmt_i", string.format("%i", 42) == "42", string.format("%i", 42), "42")
local e_str = string.format("%.2e", 123456.789)
T("tl_sfmt_e", e_str == "1.23e+05", e_str, "1.23e+05")
T("tl_sfmt_pct", string.format("100%%") == "100%", string.format("100%%"), "100%")
T("tl_sfmt_zero", string.format("%05d", 42) == "00042", string.format("%05d", 42), "00042")
T("tl_sfmt_left", string.format("%-5d|", 42) == "42   |", string.format("%-5d|", 42), "42   |")
T("tl_sfmt_plus", string.format("%+d", 42) == "+42", string.format("%+d", 42), "+42")
T("tl_sfmt_space", string.format("% d", 42) == " 42", string.format("% d", 42), " 42")
check_globals("tl_after_string_format")

-- Table operations
local t1 = {1, 2, 3}
table.insert(t1, 2, 10)
T("tl_tins", t1[2] == 10, t1[2], 10)
local t2 = {1, 2, 3, 4}
local removed = table.remove(t2, 2)
T("tl_trem", removed == 2, removed, 2)
local t3 = {10, 20, 30}
local t4 = {}
table.move(t3, 1, 3, 1, t4)
T("tl_tmove", t4[1]==10 and t4[3]==30, t4[1], 10)
local pk = table.pack(1, nil, 3)
T("tl_pack", pk.n == 3, pk.n, 3)
local ua, ub, uc = table.unpack({10, 20, 30})
T("tl_unpack", ua==10 and ub==20 and uc==30, ua, 10)
local ts = {3, 1, 4, 1, 5, 9}
table.sort(ts)
T("tl_sort", ts[1] == 1 and ts[6] == 9, ts[1], 1)
check_globals("tl_after_table_ops")

-- Coroutines
local co = coroutine.create(function(a, b)
    coroutine.yield(a + b)
    return a * b
end)
local ok1, v1 = coroutine.resume(co, 3, 4)
T("tl_co_yield", v1 == 7, v1, 7)
local ok2, v2 = coroutine.resume(co, 3, 4)
T("tl_co_ret", v2 == 12, v2, 12)
local co2 = coroutine.create(function() coroutine.yield() end)
coroutine.resume(co2)
coroutine.resume(co2)
T("tl_co_dead", coroutine.status(co2) == "dead", coroutine.status(co2), "dead")
check_globals("tl_after_coroutines")

-- pcall/xpcall
local ok3, msg3 = xpcall(function() error("boom") end, function(e) return "handled: " .. e end)
T("tl_xpcall", not ok3, tostring(ok3), "false")
local ok4, msg4 = xpcall(function() error(42) end, function(e) return e + 1 end)
T("tl_xpcall_obj", msg4 == 43, msg4, 43)
local ok5, v5 = pcall(function() return 42 end)
T("tl_pcall_ok", ok5 and v5 == 42, v5, 42)
check_globals("tl_after_pcall")

-- select
T("tl_select", select(2, "a", "b", "c") == "b", select(2, "a", "b", "c"), "b")
T("tl_select_n", select("#", "a", "b", "c") == 3, select("#", "a", "b", "c"), 3)
check_globals("tl_after_select")

-- Raw operations
local req1 = setmetatable({}, {__eq = function() return true end})
local req2 = setmetatable({}, {__eq = function() return true end})
T("tl_rawequal", not rawequal(req1, req2), tostring(rawequal(req1, req2)), "false")
local rgt = setmetatable({}, {__index = function() return 42 end})
T("tl_rawget", rawget(rgt, "x") == nil, tostring(rawget(rgt, "x")), "nil")
local rsLog = false
local rst = setmetatable({}, {__newindex = function() rsLog = true end})
rawset(rst, "x", 1)
T("tl_rawset", not rsLog, tostring(rsLog), "false")
T("tl_rawlen", rawlen({1,2,3}) == 3, rawlen({1,2,3}), 3)
check_globals("tl_after_raw_ops")

-- tonumber/tostring
T("tl_tonum", tonumber("ff", 16) == 255, tonumber("ff", 16), 255)
T("tl_tostr", tostring(nil) == "nil", tostring(nil), "nil")
check_globals("tl_after_tonumber")

-- ipairs
local ip = {10, 20, nil, 40}
local ipairs_result = {}
for i, v in ipairs(ip) do ipairs_result[i] = v end
T("tl_ipairs", #ipairs_result == 2, #ipairs_result, 2)
check_globals("tl_after_ipairs")

-- Type coercion / math.type
T("tl_int_add", math.type(3 + 4) == "integer", math.type(3 + 4), "integer")
T("tl_float_add", math.type(3 + 4.0) == "float", math.type(3 + 4.0), "float")
T("tl_int_mul", math.type(3 * 4) == "integer", math.type(3 * 4), "integer")
T("tl_int_idiv", math.type(7 // 2) == "integer", math.type(7 // 2), "integer")
T("tl_float_div", math.type(7 / 2) == "float", math.type(7 / 2), "float")
check_globals("tl_after_math_type")

-- Multiple returns
local function multi() return 1, 2, 3 end
local m1, m2, m3 = multi()
T("tl_multi", m1==1 and m2==2 and m3==3, m1, 1)
local mt_r = {multi()}
T("tl_multi_tbl", mt_r[1]==1 and mt_r[2]==2 and mt_r[3]==3, mt_r[1], 1)
local mt_adj = {multi(), "x"}
T("tl_multi_adj", mt_adj[1]==1 and mt_adj[2]=="x", mt_adj[2], "x")
check_globals("tl_after_multi_return")

-- Varargs
local function vfn(...)
    return select("#", ...)
end
T("tl_vararg", vfn(1, nil, 3) == 3, vfn(1, nil, 3), 3)
check_globals("tl_after_varargs")

-- error with non-string objects
local eok1, eerr1 = pcall(error, 42)
T("tl_error_int", eerr1 == 42, eerr1, 42)
local errT = {msg="boom"}
local eok2, eerr2 = pcall(error, errT)
T("tl_error_tbl", eerr2 == errT, tostring(eerr2 == errT), "true")
check_globals("tl_after_error_obj")

-- Patterns
local r4 = string.find("THE END", "%f[%a]%u+")
T("tl_frontier", r4 == 1, r4, 1)
local r5 = string.match("(hello (world))", "%b()")
T("tl_balanced", r5 == "(hello (world))", r5, "(hello (world))")
check_globals("tl_after_patterns")

-- Math library
T("tl_math_abs", math.abs(-5) == 5, math.abs(-5), 5)
T("tl_math_ceil", math.ceil(1.5) == 2, math.ceil(1.5), 2)
T("tl_math_floor", math.floor(1.5) == 1, math.floor(1.5), 1)
T("tl_math_sqrt", math.sqrt(9) == 3.0, math.sqrt(9), 3.0)
T("tl_math_max", math.max(1,3,2) == 3, math.max(1,3,2), 3)
T("tl_math_min", math.min(1,3,2) == 1, math.min(1,3,2), 1)
check_globals("tl_after_math_lib")

-- Integer overflow
T("tl_int_max", math.maxinteger == 0x7FFFFFFFFFFFFFFF, math.maxinteger, 0x7FFFFFFFFFFFFFFF)
T("tl_int_min", math.mininteger == -0x7FFFFFFFFFFFFFFF - 1, math.mininteger, -0x7FFFFFFFFFFFFFFF - 1)
T("tl_int_wrap", math.maxinteger + 1 == math.mininteger, math.maxinteger + 1, math.mininteger)
check_globals("tl_after_int_overflow")

-- load
local fn = load("return 42")
T("tl_load", fn() == 42, fn(), 42)
local fn2, err2 = load("invalid syntax @@!")
T("tl_load_err", fn2 == nil and type(err2) == "string", type(err2), "string")
check_globals("tl_after_load")

-- String.format edge cases
T("tl_fmt_c", string.format("%c", 65) == "A", string.format("%c", 65), "A")
T("tl_fmt_x", string.format("%x", 255) == "ff", string.format("%x", 255), "ff")
T("tl_fmt_X", string.format("%X", 255) == "FF", string.format("%X", 255), "FF")
T("tl_fmt_o", string.format("%o", 8) == "10", string.format("%o", 8), "10")
T("tl_fmt_0d", string.format("%05d", 42) == "00042", string.format("%05d", 42), "00042")
check_globals("tl_after_fmt")

-- goto at top level
local tl_goto = 0
goto tl_skip
tl_goto = 1
::tl_skip::
tl_goto = 2
saved_assert(tl_goto == 2, "toplevel goto: " .. tl_goto)
check_globals("tl_after_goto")

-- Final exhaustive global check
check_globals("FINAL")
