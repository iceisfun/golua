-- broken_fuzz_tonumber_unicode_whitespace:
-- golua treats UTF-8 Unicode whitespace as valid leading/trailing
-- whitespace in tonumber() and arithmetic coercion. Reference Lua only
-- accepts the ASCII isspace() set: \t \n \v \f \r ' '.
--
-- BROKEN: Multiple sites use Go's `strings.TrimSpace`, which calls
-- `unicode.IsSpace` and strips NBSP (U+00A0), en-space (U+2002),
-- ideographic space (U+3000), and other Unicode space code points when
-- they appear as valid UTF-8 prefixes/suffixes.
--
-- Affected sites identified in the fuzz pass:
--   stdlib/globals.go:168, :194  — tonumber
--   vm/value.go:389, :520, :572  — arithmetic / int / float coercion
--
-- Propagates to: tonumber, arithmetic on strings, string.format("%d", ...)
-- when arg is string, math.floor/tointeger on string args, string.sub /
-- string.byte / string.rep numeric args, etc. Any place that coerces a
-- string to a number is permissive about non-ASCII whitespace.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   tonumber("\xc2\xa010") -> nil
--   "\xc2\xa010" + 0  -> error "attempt to add a 'string' with a 'number'"
--   string.format("%d", "\xc2\xa05") -> error "bad argument #2 to 'format'..."
--
-- golua today:
--   tonumber("\xc2\xa010") -> 10
--   "\xc2\xa010" + 0 -> 10  (no error)
--   string.format("%d", "\xc2\xa05") -> "5"
--
-- Fix: replace strings.TrimSpace with a helper that strips ONLY bytes
-- 0x09–0x0D and 0x20.
--
-- Discovered: differential fuzz 2026-05-04 (string-library wave-2 agent).

-- 1. tonumber should reject NBSP-prefixed numerics
assert(tonumber("\xc2\xa010") == nil,
  "tonumber must not accept NBSP (U+00A0) as whitespace")
assert(tonumber("10\xc2\xa0") == nil,
  "tonumber must not accept trailing NBSP")
assert(tonumber("\xe2\x80\x8210") == nil,
  "tonumber must not accept en-space (U+2002)")
assert(tonumber("\xe3\x80\x8010") == nil,
  "tonumber must not accept ideographic space (U+3000)")

-- 2. Arithmetic coercion must reject Unicode whitespace
local s = "\xc2\xa010"
local ok = pcall(function() return s + 0 end)
assert(ok == false,
  "arithmetic on string with NBSP whitespace must raise a coercion error")

-- 3. string.format %d coercion must reject Unicode whitespace
local ok2 = pcall(string.format, "%d", "\xc2\xa05")
assert(ok2 == false,
  "string.format('%d', NBSP+digits) must error per coercion rules")

print("ok")
