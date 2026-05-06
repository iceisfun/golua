-- broken_fuzz_os_date_invalid_spec_message:
-- os.date("!%k world", t) reports the invalid spec as '%k' (truncated to 2
-- characters) instead of '%k world' (the full remainder of the format
-- string from the % to the NUL).
--
-- BROKEN: vm/default_os.go around line 339 uses
--   fmt.Errorf("invalid conversion specifier '%c%c'", '%', conv)
-- which emits exactly two characters. Reference Lua's loslib.c uses
-- lua_pushfstring(L, "invalid conversion specifier '%%%s'", conv) where
-- `conv` is a C-string starting at the '%' — glibc prints the rest of the
-- format up to the NUL terminator.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(os.date, "!%k world", 0)
--     -> false, "bad argument #1 to 'os.date' (invalid conversion specifier '%k world')"
--
-- golua today:
--   -> false, "bad argument #1 to 'os.date' (invalid conversion specifier '%k')"
--
-- Discovered: differential fuzz 2026-05-04 (os wave-3 agent).

local ok, err = pcall(os.date, "!%k world", 0)
assert(ok == false, "invalid conv spec must raise")
assert(type(err) == "string", "error must be string")
assert(err:find("'%k world'", 1, true),
  "expected full remainder '%k world' in spec error; got: " .. err)

print("ok")
