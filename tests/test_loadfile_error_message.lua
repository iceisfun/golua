-- Test: loadfile error message should not double the Go "open" prefix
local fn, err = loadfile("nonexist_xyz.lua")
assert(fn == nil)
-- Error should be "cannot open nonexist_xyz.lua: <OS error>"
-- not "cannot open nonexist_xyz.lua: open nonexist_xyz.lua: <OS error>"
-- Count how many times the filename appears - should be exactly once
local _, count = err:gsub("nonexist_xyz.lua", "")
assert(count == 1, "filename appears " .. count .. " times in error: " .. err)
assert(err:find("cannot open nonexist_xyz.lua:"), "error missing 'cannot open': " .. err)
print("OK")
