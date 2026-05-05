-- broken_fuzz_unpack_s8_panic:
-- string.unpack("s8", high-bit-set-prefix) leaks a Go runtime panic message
-- through pcall instead of raising the proper Lua error.
--
-- BROKEN: stdlib/string_pack.go:863-867 reads an 8-byte length prefix into
-- a uint64 then converts to int with `int(slen)`. When slen >= 2^63, that
-- conversion overflows to a negative int. The bounds check
--   *offset + int(slen) > len(data)
-- becomes false (negative + small < len), then the subsequent slice
--   data[*offset : *offset + int(slen)]
-- panics with "slice bounds out of range [8:7]" — a Go-internals message.
--
-- pcall catches the panic, but the error string ("runtime error: slice
-- bounds out of range") leaks Go internals to Lua code. Reference Lua
-- (5.4.8 and 5.5.0) both raise the structured error
--   bad argument #2 to 'string.unpack' (data string too short)
-- which is what callers depend on for error containment.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(string.unpack, "s8", "\xff\xff\xff\xff\xff\xff\xff\xff")
--     -> false, "bad argument #2 to 'string.unpack' (data string too short)"
--
-- golua today:
--   pcall(string.unpack, "s8", "\xff\xff\xff\xff\xff\xff\xff\xff")
--     -> false, "runtime error: slice bounds out of range [8:7]"
--
-- Fix sketch: after reading slen, check `slen > uint64(len(data) - *offset)`
-- before any int(slen) arithmetic.
--
-- Discovered: differential fuzz 2026-05-04 (string.pack wave-2 agent).
-- Suspect site: stdlib/string_pack.go:863-867.

local ok, err = pcall(string.unpack, "s8", "\xff\xff\xff\xff\xff\xff\xff\xff")
assert(ok == false, "unpack should fail on too-short data")
assert(type(err) == "string", "error must be a string")
assert(err:find("data string too short"),
  "expected 'data string too short' Lua error; got Go panic leak: " .. err)
assert(not err:find("runtime error"),
  "Go runtime panic message must not leak through pcall: " .. err)

print("ok")
