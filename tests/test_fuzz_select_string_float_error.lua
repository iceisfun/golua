-- broken_fuzz_select_string_float_error:
-- select("2.5", ...) reports "number expected, got string" instead of
-- "number has no integer representation". The 1st arg is a string that
-- parses as a non-integer float, and reference Lua's coercion path
-- diagnoses that as "no integer representation"; golua takes a wrong
-- branch in its error reporter.
--
-- BROKEN: stdlib/globals.go around lines 709-718. After idx.ToInt() fails,
-- the code only checks idx.IsNumber() (false for string type) and falls
-- to the "got <typename>" path. It should first try idx.ToNumber() to
-- detect string-encoded numerics that don't have an integer representation
-- and emit the canonical reference error.
--
-- Affects every call to select() where the first arg is a string-encoded
-- non-integer value: "2.5", "0x1.8p0", "1e1.5", etc.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(select, "2.5", "a", "b")
--     -> false, "bad argument #1 to 'select' (number has no integer representation)"
--
-- golua today:
--   -> false, "bad argument #1 to 'select' (number expected, got string)"
--
-- Discovered: differential fuzz 2026-05-04 (vararg+select wave-3 agent).

local cases = {"2.5", "0x1.8p0"}
for _, s in ipairs(cases) do
  local ok, err = pcall(select, s, "a", "b")
  assert(ok == false, "select must fail")
  assert(type(err) == "string", "error must be string")
  assert(err:find("no integer representation"),
    "expected 'no integer representation' for select " .. s ..
    "; got: " .. err)
end

print("ok")
