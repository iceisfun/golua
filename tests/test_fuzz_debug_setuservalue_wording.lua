-- broken_fuzz_debug_setuservalue_wording:
-- debug.setuservalue argument-error message wording diverges from
-- reference Lua in two ways: emits "full userdata expected" instead of
-- "userdata expected"; uses arg.Type() (yielding "got nil") instead of
-- detecting absent args (which should yield "got no value").
--
-- BROKEN: stdlib/debug.go around line 721. Other stdlib functions
-- correctly use the gotDesc(v, n) helper from stdlib/globals.go which
-- distinguishes nil-from-absent. setuservalue's message also has the
-- "full userdata" qualifier which reference Lua omits.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(debug.setuservalue, {}, 1)
--     -> false, "bad argument #1 to 'debug.setuservalue' (userdata expected, got table)"
--   pcall(debug.setuservalue)  -- no args
--     -> false, "... (userdata expected, got no value)"
--
-- golua today:
--   "(full userdata expected, got table)"  /  "(full userdata expected, got nil)"
--
-- Discovered: differential fuzz 2026-05-04 (debug wave-3 agent).

local ok1, err1 = pcall(debug.setuservalue, {}, 1)
assert(ok1 == false, "must fail with non-userdata")
assert(err1:find("(userdata expected, got table)", 1, true),
  "expected '(userdata expected, got table)'; got: " .. err1)
assert(not err1:find("full userdata"),
  "should not say 'full userdata'; got: " .. err1)

local ok2, err2 = pcall(debug.setuservalue)
assert(ok2 == false, "must fail with no args")
assert(err2:find("got no value", 1, true),
  "expected 'got no value' for missing arg; got: " .. err2)

print("ok")
