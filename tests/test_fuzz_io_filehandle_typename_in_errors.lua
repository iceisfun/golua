-- broken_fuzz_io_filehandle_typename_in_errors:
-- Error messages that mention an io file handle's type print "userdata"
-- instead of "FILE*" (reference Lua uses the __name metafield).
--
-- BROKEN: stdlib/io.go at multiple sites uses arg.Type() (bare Lua type
-- name) instead of v.ObjTypeName(arg) (which respects __name = "FILE*").
-- Sites identified: ~536, 740, 1042, 1229, 1372.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(io.stdout.write, io.stdout, io.stdin)
--     -> false, "bad argument #2 to 'write' (string expected, got FILE*)"
--
-- golua today:
--   -> "(string expected, got userdata)"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local ok, err = pcall(io.stdout.write, io.stdout, io.stdin)
assert(ok == false, "write with file-handle arg must fail")
assert(type(err) == "string", "error must be a string")
assert(err:find("FILE*", 1, true),
  "expected 'FILE*' typename in error; got: " .. err)

print("ok")
