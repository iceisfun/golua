-- Test that syntax errors with high bytes report raw first byte, not decoded codepoint

-- UTF-8 encoded é (U+00E9) is bytes 0xC3 0xA9
-- Lua 5.4 reports the first byte: <\195> (0xC3 = 195)
-- GoLua was incorrectly reporting the codepoint: <\233> (0xE9 = 233)
local ok, err = load("\xC3\xA9")
assert(not ok)
assert(err:find("<\\195>"), "should report first byte \\195, got: " .. err)

-- UTF-8 encoded NBSP (U+00A0) is bytes 0xC2 0xA0
-- Should report first byte \194, not codepoint \160
ok, err = load("\xC2\xA0")
assert(not ok)
assert(err:find("<\\194>"), "should report first byte \\194, got: " .. err)

-- Single high byte (not valid UTF-8) should report that byte
ok, err = load("\x80")
assert(not ok)
assert(err:find("<\\128>"), "should report byte \\128, got: " .. err)

-- Another multi-byte: U+00FF (bytes 0xC3 0xBF)
-- Should report \195, not \255
ok, err = load("\xC3\xBF")
assert(not ok)
assert(err:find("<\\195>"), "should report first byte \\195, got: " .. err)

print("OK")
