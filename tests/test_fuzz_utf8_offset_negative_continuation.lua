-- test_fuzz_utf8_offset_negative_continuation:
-- utf8.offset(s, -1) must reject an initial position that is a continuation byte.

local ok, err = pcall(utf8.offset, "\128", -1)
assert(ok == false,
  "utf8.offset should error for negative offset starting on continuation byte")
assert(type(err) == "string" and err:find("continuation byte", 1, true),
  "expected continuation-byte error, got: " .. tostring(err))

print("ok")
