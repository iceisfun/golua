-- Test that loadfile() handles UTF-8 BOM prefix (like Lua 5.4)
-- The BOM is bytes 0xEF 0xBB 0xBF

-- Write a temp file with BOM prefix
local tmpname = os.tmpname()
local f = io.open(tmpname, "wb")
f:write("\xEF\xBB\xBF")
f:write("return 42\n")
f:close()

-- loadfile() should strip BOM
local chunk, err = loadfile(tmpname)
assert(chunk, "loadfile() with BOM should succeed: " .. tostring(err))
assert(chunk() == 42, "loadfile() with BOM should return correct value")

-- Write a file with BOM + shebang
local f2 = io.open(tmpname, "wb")
f2:write("\xEF\xBB\xBF#!/usr/bin/env lua\nreturn 99\n")
f2:close()

local chunk2, err2 = loadfile(tmpname)
assert(chunk2, "loadfile() with BOM + shebang should succeed: " .. tostring(err2))
assert(chunk2() == 99, "loadfile() with BOM + shebang should return correct value")

os.remove(tmpname)

print("OK")
