-- test_fuzz_pack_missing_float_arg_nil:
-- Missing float pack arguments are reported as nil by lua5.5.0, not "no value".

local ok, err = pcall(string.pack, "f")
assert(ok == false, "string.pack('f') without value should fail")
assert(err == "bad argument #2 to 'string.pack' (number expected, got nil)",
  "unexpected error: " .. tostring(err))

print("ok")
