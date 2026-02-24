-- string_basic.lua: Regression tests for string library functions.
-- These tests lock in correct, passing behavior. Any failure is a regression.

local function assert_eq(a, b, msg)
    if a ~= b then
        error((msg or "assertion failed") .. ": expected " .. tostring(b) .. ", got " .. tostring(a), 2)
    end
end

--------------------------------------------------------------------------------
-- string.match: basic capture behavior
--------------------------------------------------------------------------------

-- Single capture
assert_eq(string.match("hello123", "%d+"), "123")

-- Two captures
local a, b = string.match("hello123", "(%a+)(%d+)")
assert_eq(a, "hello")
assert_eq(b, "123")

-- No match returns nil
assert_eq(string.match("hello", "%d+"), nil)

-- Anchored match at start
assert_eq(string.match("hello", "^%a+"), "hello")
assert_eq(string.match("123hello", "^%a+"), nil)

-- Anchored match at end
assert_eq(string.match("hello", "%a+$"), "hello")

-- Init position (3rd arg)
assert_eq(string.match("abcabc", "%a+", 4), "abc")

-- Character classes
assert_eq(string.match("test 123", "%d+"), "123")
assert_eq(string.match("test 123", "%a+"), "test")
assert_eq(string.match("  hello  ", "%S+"), "hello")

-- Dot matches any character
assert_eq(string.match("abc", "."), "a")
assert_eq(string.match("abc", ".."), "ab")

-- Quantifiers
assert_eq(string.match("aabbb", "a+"), "aa")
assert_eq(string.match("aabbb", "b+"), "bbb")
assert_eq(string.match("aabbb", "b*"), "")  -- greedy * can match 0
assert_eq(string.match("abc", "a?b"), "ab")

-- Character sets
assert_eq(string.match("hello", "[helo]+"), "hello")
assert_eq(string.match("abc123", "[%d]+"), "123")
assert_eq(string.match("abc", "[a-c]+"), "abc")
assert_eq(string.match("abc", "[^a]+"), "bc")

-- Nested captures
local x, y = string.match("(hello)", "%((%a+)%)")
assert_eq(x, "hello")

--------------------------------------------------------------------------------
-- string.gsub: string replacement
--------------------------------------------------------------------------------

-- Simple literal replacement
assert_eq(string.gsub("hello world", "world", "lua"), "hello lua")

-- Pattern replacement
assert_eq(string.gsub("abc123def", "%d+", "NUM"), "abcNUMdef")

-- Multiple replacements
local r1, n1 = string.gsub("aaa", "a", "b")
assert_eq(r1, "bbb")
assert_eq(n1, 3)

-- Limited replacements (4th arg)
local r2, n2 = string.gsub("aaa", "a", "b", 2)
assert_eq(r2, "bba")
assert_eq(n2, 2)

-- Replacement with captures (%1, %2)
assert_eq(string.gsub("hello world", "(%w+)", "[%1]"), "[hello] [world]")

-- %0 = whole match
assert_eq(string.gsub("abc", "%a", "(%0)"), "(a)(b)(c)")

-- %% = literal percent
assert_eq(string.gsub("abc", "%a", "%%"), "%%%")

-- No match = unchanged
assert_eq(string.gsub("hello", "%d", "x"), "hello")

-- Empty pattern (inserts at each position)
assert_eq(string.gsub("abc", "", "x"), "xaxbxcx")
assert_eq(string.gsub("", "", "x"), "x")

-- Empty pattern with limit
local r3, n3 = string.gsub("abc", "", "x", 2)
assert_eq(r3, "xaxbc")
assert_eq(n3, 2)

--------------------------------------------------------------------------------
-- string.gsub: function replacement
--------------------------------------------------------------------------------

-- Basic function replacement (no captures)
local r4, n4 = string.gsub("123", "%d", function(d) return "(" .. d .. ")" end)
assert_eq(r4, "(1)(2)(3)")
assert_eq(n4, 3)

-- Function replacement with pattern captures
local r5, n5 = string.gsub("hello world", "(%w+)", function(w) return w:upper() end)
assert_eq(r5, "HELLO WORLD")
assert_eq(n5, 2)

-- Multiple captures passed to function
local r6 = string.gsub("2025-01-15", "(%d+)-(%d+)-(%d+)", function(y,m,d)
    return d .. "/" .. m .. "/" .. y
end)
assert_eq(r6, "15/01/2025")

-- Function returning nil keeps original match
local r7 = string.gsub("abc", "%a", function(c)
    if c == "b" then return nil end
    return c:upper()
end)
assert_eq(r7, "AbC")

-- Function returning false keeps original match
local r8 = string.gsub("abc", "%a", function(c)
    if c == "b" then return false end
    return c:upper()
end)
assert_eq(r8, "AbC")

-- Function returning false with MULTIPLE captures preserves whole match
-- Bug: captures[0] was returned (first capture) instead of the whole match
local r8b = string.gsub("a1b2c3", "(%a)(%d)", function(letter, digit)
    if tonumber(digit) > 1 then return letter .. "[" .. digit .. "]" end
    return false
end)
assert_eq(r8b, "a1b[2]c[3]", "gsub false return with multi-captures")

-- Function returning nil with MULTIPLE captures preserves whole match
local r8c = string.gsub("x9y8", "(%a)(%d)", function(letter, digit)
    if tonumber(digit) > 8 then return letter:upper() .. digit end
    return nil
end)
assert_eq(r8c, "X9y8", "gsub nil return with multi-captures")

-- Function with max replacement
local r9, n9 = string.gsub("aaa", "a", function() return "b" end, 2)
assert_eq(r9, "bba")
assert_eq(n9, 2)

-- Function returning a number (auto-converted to string)
local r10 = string.gsub("abc", "%a", function() return 1 end)
assert_eq(r10, "111")

-- Empty pattern with function replacement
local r11 = string.gsub("abc", "", function() return "x" end)
assert_eq(r11, "xaxbxcx")

--------------------------------------------------------------------------------
-- string.gsub: table replacement
--------------------------------------------------------------------------------

-- Basic table lookup
local t = {hello = "HI", world = "EARTH"}
local r12 = string.gsub("hello world", "(%w+)", t)
assert_eq(r12, "HI EARTH")

-- Table miss keeps original
local t2 = {hello = "HI"}
local r13 = string.gsub("hello world", "(%w+)", t2)
assert_eq(r13, "HI world")

-- Table value false keeps original
local t3 = {hello = "HI", world = false}
local r14 = string.gsub("hello world", "(%w+)", t3)
assert_eq(r14, "HI world")

--------------------------------------------------------------------------------
-- string.gsub: return value (count)
--------------------------------------------------------------------------------

local _, count1 = string.gsub("abc", "%a", "x")
assert_eq(count1, 3)

local _, count2 = string.gsub("abc", "%d", "x")
assert_eq(count2, 0)

local _, count3 = string.gsub("abc", "", "x")
assert_eq(count3, 4)  -- before each char + at end
