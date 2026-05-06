-- broken_fuzz_require_numeric_path:
-- package.searchpath() raises a type error when called with a numeric
-- path argument, instead of coercing the number to its string form like
-- reference Lua. Same defect underlies require() with package.path /
-- package.cpath set to a number — reference coerces via lua_tostring at
-- lookup time.
--
-- BROKEN: stdlib/package.go around lines 280 (Lua-file searcher) and
-- 333 (C-file searcher), and the package.searchpath validator. They
-- check pathVal.IsString() and reject numbers; should also accept
-- numbers via the standard string coercion (use the same getString-style
-- helper other stdlib uses).
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   package.searchpath("any", 42)
--     -> succeeds (returns nil + tried paths, OR finds a file named "42"
--        if one exists in cwd — depends on filesystem state).
--   The KEY behavior is that searchpath does NOT raise a type error for
--   numeric path; it coerces the number to "42" and proceeds.
--
-- golua today:
--   raises a type error like "'package.path' must be a string"
--
-- Discovered: differential fuzz 2026-05-04 (package wave-3 agent).

local ok, err = pcall(package.searchpath,
  "definitely_not_a_module_zzz_aaa_123_xyz", 42)
assert(ok == true,
  "package.searchpath must not raise type error on numeric path " ..
  "(reference coerces 42 -> '42'); got error: " .. tostring(err))

-- Also confirm directly setting package.path = number doesn't break require's
-- searcher with a type error. require may still fail to find the module — that's
-- fine; the failure must be "module not found" or success, NEVER a type error.
package.path = 42
local rok, rerr = pcall(require, "definitely_not_a_module_zzz_aaa_123_xyz")
if rok == false then
  assert(not tostring(rerr):find("must be a string"),
    "require with numeric package.path must not raise type error; got: " ..
    tostring(rerr))
end

print("ok")
