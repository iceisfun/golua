-- Test that package.loaded[mod] = false does NOT count as cached.
-- Lua 5.4: only non-nil values in package.loaded are treated as cached.
-- Setting package.loaded["testmod"] = false should cause require to search
-- for the module (and fail if not found), not return false.

package.loaded["nonexistent_mod_xyz"] = false
local ok, result = pcall(require, "nonexistent_mod_xyz")
assert(ok == false, "expected pcall to fail, got ok=" .. tostring(ok))
assert(type(result) == "string", "expected error string, got " .. type(result))
assert(result:find("not found"), "expected 'not found' in error, got: " .. result)

print("PASS")
