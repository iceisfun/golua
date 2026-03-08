-- Test: Non-ASCII multi-byte chars in "near" should use first byte, not codepoint
-- Bug: GoLua decodes UTF-8 and shows the codepoint value in <\NNN> format.
-- Lua 5.4 shows the first byte value since it is byte-oriented.
-- BOM (\xEF\xBB\xBF = U+FEFF):
--   Lua 5.4: near '<\239>'   (first byte 0xEF = 239)
--   GoLua:   near '<\65279>' (codepoint U+FEFF = 65279)

local _, err = load("\xEF\xBB\xBFreturn 1")
assert(err:find("<\\239>"), "BOM byte: got: " .. tostring(err))

print("PASS")
