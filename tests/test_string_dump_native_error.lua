-- Bug: string.dump on a native function gives misleading error message.
-- Says "function expected, got function" instead of something like
-- "unable to dump given function".

local ok, err = pcall(string.dump, print)
assert(not ok, "string.dump(print) should error")
-- The error message should NOT say "function expected, got function"
assert(not err:find("function expected, got function"),
  "error message should not say 'function expected, got function': " .. err)
-- Should say something meaningful
assert(err:find("unable to dump") or err:find("cannot dump") or err:find("dump"),
  "error should indicate function can't be dumped: " .. err)

print("PASS")
