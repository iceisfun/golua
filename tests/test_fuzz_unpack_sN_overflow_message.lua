-- broken_fuzz_unpack_sN_overflow_message:
-- string.unpack("sN", ...) for N > 8 reports "data string too short" when
-- it should report "N-byte integer does not fit into Lua Integer".
--
-- BROKEN: stdlib/string_pack.go around lines 846-861 (unpackSizedString
-- prefixSize > 8 path) raises "data string too short" when extension bytes
-- (bytes 9..N) are non-zero. Reference Lua diagnoses this as the
-- length-prefix-too-large case and raises
--   "N-byte integer does not fit into Lua Integer"
-- via unpackInt's existing path (string_pack.go around line 718).
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   pcall(string.unpack, "s12",
--         "\x05\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00xxx")
--     -> false, "12-byte integer does not fit into Lua Integer"
--
-- golua today:
--   -> false, "bad argument #2 to 'string.unpack' (data string too short)"
--
-- Affects sN for N in 9..16. Wrong error class — callers can't
-- distinguish "the data was truncated" from "the encoded length exceeded
-- Lua's integer range".
--
-- Discovered: differential fuzz 2026-05-04 (string.pack wave-2 agent).

local data = "\x05\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00xxx"
local ok, err = pcall(string.unpack, "s12", data)
assert(ok == false, "unpack should fail")
assert(type(err) == "string", "error must be a string")
assert(err:find("does not fit into Lua Integer"),
  "expected 'does not fit into Lua Integer' diagnosis; got: " .. err)

print("ok")
