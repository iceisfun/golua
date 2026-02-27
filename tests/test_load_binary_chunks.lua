-- Bug 1: load() cannot load binary chunks produced by string.dump()
-- Bug 2: load() with binary chunk in mode "t" gives wrong error message
-- Bug 3: load() incorrectly strips shebang (#!) from string arguments

-- Test 1: load(string.dump(f)) should work for round-tripping
local f = function() return 42 end
local dumped = string.dump(f)
assert(type(dumped) == "string", "string.dump should return string")
local g, err = load(dumped)
assert(g, "load(string.dump(f)) should succeed, got error: " .. tostring(err))
assert(g() == 42, "loaded function should return 42")

-- Test 2: binary chunk with mode "t" should give specific error
local f2 = function() return 99 end
local bin = string.dump(f2)
local g2, err2 = load(bin, "test", "t")
assert(g2 == nil, "should reject binary chunk in text mode")
assert(err2:find("binary chunk"), "error should mention 'binary chunk', got: " .. tostring(err2))

-- Test 3: shebang should NOT be stripped from string args to load()
local s = "#!/usr/bin/lua\nreturn 43"
local f3, err3 = load(s)
assert(f3 == nil, "load() should NOT strip shebang from string args")
assert(err3:find("unexpected symbol") or err3:find("#"),
  "error should mention unexpected symbol, got: " .. tostring(err3))

print("PASS")
