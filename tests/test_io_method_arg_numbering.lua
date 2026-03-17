-- File method errors should count 'self' as arg #1.
-- In C Lua, f:setvbuf("bad") reports "bad argument #2 to '?'"
-- because self is arg #1 and the mode string is arg #2.

local tmp = os.tmpname()
local f = assert(io.open(tmp, "w"))

-- setvbuf: invalid mode should be arg #2
local ok1, err1 = pcall(f.setvbuf, f, "bad")
assert(not ok1, "setvbuf should error on invalid mode")
assert(err1:find("#2"), "setvbuf error should reference arg #2, got: " .. err1)

-- seek: invalid whence should be arg #2
local ok2, err2 = pcall(f.seek, f, "bad")
assert(not ok2, "seek should error on invalid whence")
assert(err2:find("#2"), "seek error should reference arg #2, got: " .. err2)

-- read: invalid format type should be arg #2
local ok3, err3 = pcall(f.read, f, true)
assert(not ok3, "read should error on boolean format")
assert(err3:find("#2"), "read error should reference arg #2, got: " .. err3)

f:close()
os.remove(tmp)
