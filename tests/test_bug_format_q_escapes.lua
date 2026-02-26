-- Bug: string.format("%q", ...) doesn't properly escape control characters.
-- Lua 5.4 %q produces a string that, when read back by the Lua parser, gives
-- the original. Newlines become \<newline>, control chars use \N decimal form.

-- Newline: Lua 5.4 outputs backslash followed by literal newline
local q_nl = string.format("%q", "hello\nworld")
-- The result should be loadable and reproduce the original
local f_nl = load("return " .. q_nl)
assert(f_nl, "%q newline not loadable: " .. q_nl)
assert(f_nl() == "hello\nworld", "%q newline round-trip failed")

-- Tab: Lua 5.4 escapes as \9
local q_tab = string.format("%q", "hello\tworld")
assert(not q_tab:find("\t"), "%q should escape tab, got literal tab: " .. q_tab)
local f_tab = load("return " .. q_tab)
assert(f_tab and f_tab() == "hello\tworld", "%q tab round-trip failed")

-- Carriage return: Lua 5.4 escapes as \13
local q_cr = string.format("%q", "hello\rworld")
assert(not q_cr:find("\r"), "%q should escape CR, got literal CR")
local f_cr = load("return " .. q_cr)
assert(f_cr and f_cr() == "hello\rworld", "%q CR round-trip failed")

-- Null byte: Lua 5.4 escapes as \0
local q_null = string.format("%q", "hello\0world")
local f_null = load("return " .. q_null)
assert(f_null, "%q null not loadable")
assert(f_null() == "hello\0world", "%q null round-trip failed")

-- Control char (byte 1): should be escaped as \1
local q_ctrl = string.format("%q", string.char(1))
assert(#q_ctrl > 2, "%q of \\1 should not be empty string: got " .. q_ctrl)
local f_ctrl = load("return " .. q_ctrl)
assert(f_ctrl, "%q ctrl-A not loadable: " .. q_ctrl)
assert(f_ctrl() == string.char(1), "%q ctrl-A round-trip failed")

-- All control chars (0-31) should round-trip
for i = 0, 31 do
    local ch = string.char(i)
    local q = string.format("%q", ch)
    local f = load("return " .. q)
    assert(f, string.format("%%q of byte %d not loadable: %s", i, q))
    assert(f() == ch, string.format("%%q of byte %d round-trip failed", i))
end

-- Byte 255: %q should pass it through (not a control char)
local q_255 = string.format("%q", string.char(255))
assert(#q_255 > 2, "%q of byte 255 should produce non-empty quoted string")

-- Backslash: should be escaped
local q_bs = string.format("%q", [[hello\world]])
local f_bs = load("return " .. q_bs)
assert(f_bs and f_bs() == [[hello\world]], "%q backslash round-trip failed")

-- Double quote: should be escaped
local q_dq = string.format("%q", 'hello"world')
local f_dq = load("return " .. q_dq)
assert(f_dq and f_dq() == 'hello"world', "%q double-quote round-trip failed")

-- Guard: normal strings should still work
local q_normal = string.format("%q", "hello world")
assert(q_normal == '"hello world"', "%q normal string: " .. q_normal)
