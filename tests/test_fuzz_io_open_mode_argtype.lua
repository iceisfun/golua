-- broken_fuzz_io_open_mode_argtype:
-- io.open(path, <non-string mode>) reports "(invalid mode)" instead of
-- reference Lua's "(string expected, got <typename>)".
--
-- BROKEN: stdlib/io.go around line 540 does mode = v.Get(2).AsString() for
-- any non-nil arg, getting "" for non-strings, then tripping the
-- "invalid mode" path. Should type-check explicitly first.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   io.open("/tmp/x", true) -> nil + ... but pcall raises:
--     "bad argument #2 to 'open' (string expected, got boolean)"
--   io.open("/tmp/x", {})   -> "string expected, got table"
--
-- golua today:
--   -> "bad argument #2 to 'io.open' (invalid mode '')"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()

local ok1, err1 = pcall(io.open, p, true)
os.remove(p)
assert(ok1 == false, "io.open with bool mode must fail")
assert(err1:find("string expected"),
  "expected 'string expected' for bool mode; got: " .. err1)

local ok2, err2 = pcall(io.open, p, {})
assert(ok2 == false, "io.open with table mode must fail")
assert(err2:find("string expected"),
  "expected 'string expected' for table mode; got: " .. err2)

print("ok")
