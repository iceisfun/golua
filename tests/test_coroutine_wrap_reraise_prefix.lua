-- Regression: coroutine.wrap re-raises a string error by UNCONDITIONALLY
-- prepending the caller location (luaL_where(L,1), Lua's luaB_auxwrap). When
-- the inner error's origin line equals the wrap call site, the source:line:
-- prefix legitimately appears TWICE. golua previously deduped it away via the
-- AddCallerLocation HasPrefix guard.

-- Inner error and the w() call site are both on line 1 of the chunk, so the
-- caller-location prefix and the error's own prefix coincide -> doubled.
local chunk = 'local w = coroutine.wrap(function() error("we") end); return w()'
local f = assert(load(chunk, "=C"))
local ok, e = pcall(f)
assert(not ok)
assert(e == "C:1: C:1: we", "expected doubled prefix, got: " .. tostring(e))

-- When origin line differs from the wrap call site, both distinct prefixes show.
local chunk2 = table.concat({
  'local w = coroutine.wrap(function()',     -- line 1
  '  error("oops")',                          -- line 2: error origin
  'end)',                                     -- line 3
  'return w()',                               -- line 4: wrap call site
}, "\n")
local f2 = assert(load(chunk2, "=C"))
local ok2, e2 = pcall(f2)
assert(not ok2)
assert(e2 == "C:4: C:2: oops", "expected distinct prefixes, got: " .. tostring(e2))

-- error(msg, 0) suppresses the inner prefix; wrap still adds the caller one.
local chunk3 = 'local w = coroutine.wrap(function() error("bare", 0) end); return w()'
local ok3, e3 = pcall(assert(load(chunk3, "=C")))
assert(not ok3)
assert(e3 == "C:1: bare", "expected single wrap prefix, got: " .. tostring(e3))

-- Non-string error objects propagate unchanged (no prefix prepended).
local chunk4 = 'local w = coroutine.wrap(function() error({tag="x"}) end); return w()'
local ok4, e4 = pcall(assert(load(chunk4, "=C")))
assert(not ok4)
assert(type(e4) == "table" and e4.tag == "x", "table error object must pass through unchanged")

print("PASS")
