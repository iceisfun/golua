-- Bug: load() strips shebang (#!) from string arguments.
-- In Lua 5.4, shebang stripping only happens for file loading,
-- not for load() with string arguments.

-- Test 1: string with shebang should fail (not be stripped)
local s = "#!/usr/bin/lua\nreturn 43"
local f, err = load(s)
assert(f == nil, "load() should NOT strip shebang from string args, but it loaded successfully")

-- Test 2: verify the error is about the # character
assert(err:find("unexpected symbol") or err:find("#"),
  "error should mention unexpected symbol near #, got: " .. tostring(err))

