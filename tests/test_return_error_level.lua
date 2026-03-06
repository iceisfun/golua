-- return error() in tail call position must preserve source location
-- Bug: native tail calls didn't push a frame, so error() couldn't find
-- the calling function's source location

-- Helper
local function ctx(err) return err:match("^(.-):%s") end

-- return error("msg") should include file:line
local function f1() return error("msg") end
local ok, err = pcall(f1)
assert(not ok)
assert(err:find(":.*: msg"), "return error(): expected file:line prefix, got: " .. err)

-- return assert(nil) should include file:line
local function f2() return assert(nil) end
local ok2, err2 = pcall(f2)
assert(not ok2)
assert(err2:find(":.*: assertion failed!"), "return assert(): expected file:line prefix, got: " .. err2)

-- return error("msg", 0) should NOT have prefix (level 0)
local function f3() return error("msg", 0) end
local ok3, err3 = pcall(f3)
assert(not ok3)
assert(err3 == "msg", "error(msg, 0): expected no prefix, got: " .. err3)

-- return error("msg", 2) with tail call: level 2 goes past available frames
local function f4() return error("msg", 2) end
local ok4, err4 = pcall(f4)
assert(not ok4)
assert(err4 == "msg", "error(msg, 2) tail call: expected no prefix, got: " .. err4)

print("PASS")
