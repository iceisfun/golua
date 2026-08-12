-- Test file for round 20 fixes

-- Fix 1: package.config missing trailing newline
assert(string.byte(package.config, #package.config) == 10,
  "package.config should end with newline (byte 10)")

-- Fix 2: package.loaded["_G"] not set
assert(package.loaded["_G"] == _G,
  "package.loaded should contain _G")

-- Fix 3: loadfile/dofile error messages use title-case OS error descriptions.
-- Use a relative nonexistent name so the path resolves inside the jail root
-- (hardcoded /tmp breaks under a sandboxed TMPDIR); the file does not exist so
-- loadfile surfaces a real ENOENT with the title-case description.
local _, e = loadfile("nonexistent_XYZ_golua_test.lua")
assert(e:find("No such file"), "loadfile error should have title-case 'No such file', got: " .. tostring(e))

-- Fix 4: string.pack/unpack/packsize use qualified names in errors (matching Lua 5.4)
local ok, e = pcall(string.pack, "b", 200)
assert(not ok)
assert(e:find("'string%.pack'"), "pack error should say 'string.pack', got: " .. tostring(e))

local ok2, e2 = pcall(string.unpack, "b", "")
assert(not ok2)
assert(e2:find("'string%.unpack'"), "unpack error should say 'string.unpack', got: " .. tostring(e2))

local ok3, e3 = pcall(string.packsize, "z")
assert(not ok3)
assert(e3:find("'string%.packsize'"), "packsize error should say 'string.packsize', got: " .. tostring(e3))

print("PASS")
