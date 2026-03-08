-- Test os.rename
-- This test requires FullIoProvider (io_ prefix triggers it)

-- Create a temp file, rename it, verify
local tmpname = os.tmpname()
local f = io.open(tmpname, "w")
f:write("rename test")
f:close()

local newname = tmpname .. ".renamed"
local ok = os.rename(tmpname, newname)
assert(ok == true, "os.rename should return true on success")

-- Verify the renamed file exists and has correct content
local f2 = io.open(newname, "r")
assert(f2 ~= nil, "renamed file should exist")
local content = f2:read("a")
f2:close()
assert(content == "rename test", "content mismatch after rename")

-- Clean up
os.remove(newname)

-- Rename a non-existent file should return nil + error
local ok2, err2 = os.rename("/tmp/nonexistent_golua_test_file_xyz", "/tmp/other_xyz")
assert(ok2 == nil, "os.rename of non-existent file should return nil")
assert(type(err2) == "string", "os.rename error should be a string")

print("PASS")
