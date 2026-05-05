-- broken_fuzz_utf8_codes_lax_first_byte:
-- utf8.codes(s, lax) skips first-byte validation when lax=true.
--
-- BROKEN: stdlib/utf8.go around line 283 gates the first-byte rune-start
-- check with `!lax && len(s) > 0 && !utf8.RuneStart(s[0])`. Reference Lua
-- (lutf8lib.c:261) does this check unconditionally — lax mode does NOT skip
-- it. Result: golua's `utf8.codes("\x80", true)` succeeds at setup (returns
-- the iterator triple) where reference raises  bad argument #1 to
-- 'utf8.codes' (invalid UTF-8 code).
--
-- The loop body in golua eventually errors with "invalid UTF-8 code" but
-- the message lacks the "bad argument #1 to 'codes'" prefix that reference
-- emits.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(utf8.codes, "\x80", true)
--     -> false, "bad argument #1 to 'utf8.codes' (invalid UTF-8 code)"
--
-- golua today:
--   pcall(utf8.codes, "\x80", true)
--     -> true, <function>   (validation skipped at setup)
--
-- Discovered: differential fuzz 2026-05-04 (utf8 wave-2 agent).
-- Suspect: drop the `!lax` gate at stdlib/utf8.go:283.

local s = "\x80"

-- Setup-time validation (the actual bug).
local ok, err = pcall(utf8.codes, s, true)
assert(ok == false,
  "utf8.codes setup should error on invalid first byte even in lax mode")
assert(type(err) == "string" and err:find("invalid UTF%-8 code"),
  "expected 'invalid UTF-8 code' in error; got " .. tostring(err))
assert(err:find("bad argument"),
  "expected 'bad argument' prefix in setup-time error; got " .. err)

print("ok")
