-- test_fuzz_ipairs_nil_tforcall_prefix:
-- A runtime error raised inside a C iterator driven by a generic-for loop
-- (e.g. the ipairs iterator indexing a nil "table") must NOT be prefixed
-- with the for-loop's source:line. The error is raised inside a C function,
-- so Lua 5.5's luaG_runerror attaches no position. golua used to add the
-- caller's location on panic recovery (AddCallerLocation), wrongly prefixing
-- it; that is now applied only to luaL_error-style string panics.
--
-- Discovered: differential scout 2026-05-20 (control-flow agent).

local ok, err = pcall(function() for i, v in ipairs(nil) do end end)
assert(not ok, "ipairs(nil) iteration must fail")
assert(err:find("attempt to index a nil value", 1, true), "got: " .. tostring(err))
assert(not err:find("^[^ ]*:%d+:"),
  "C-iterator runtime error must carry no source:line prefix, got: " .. tostring(err))

-- A bad-argument error from a stdlib function called from Lua DOES still get
-- the caller location prefix (luaL_where(L,1) behavior).
local ok2, err2 = pcall(function() return string.rep("x", "notnum") end)
assert(not ok2)
assert(err2:find("^[^ ]*:%d+: bad argument"),
  "stdlib argerror must keep its caller source:line prefix, got: " .. tostring(err2))

print("ok")
