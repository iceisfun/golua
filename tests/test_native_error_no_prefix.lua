-- Tests that errors from native/C-level functions don't include file:line prefix.
-- In Lua 5.4, errors thrown via luaG_runerror from C functions don't include
-- file:line because the current frame is a C function, not a Lua frame.

-- rawset with nil key
local function test_rawset_nil() rawset({}, nil, 1) end
local ok, err = pcall(test_rawset_nil)
assert(not ok)
assert(err == "table index is nil", "rawset nil: got: " .. tostring(err))

-- rawset with NaN key
local function test_rawset_nan() rawset({}, 0/0, 1) end
local ok2, err2 = pcall(test_rawset_nan)
assert(not ok2)
assert(err2 == "table index is NaN", "rawset NaN: got: " .. tostring(err2))

-- table.unpack with nil
local function test_unpack_nil() return table.unpack(nil) end
local ok3, err3 = pcall(test_unpack_nil)
assert(not ok3)
assert(err3 == "attempt to get length of a nil value", "unpack nil: got: " .. tostring(err3))

-- table.sort with incompatible types
local function test_sort_mixed() table.sort({1, "a"}) end
local ok4, err4 = pcall(test_sort_mixed)
assert(not ok4)
assert(err4 == "attempt to compare string with number", "sort: got: " .. tostring(err4))

-- Verify no file:line prefix
assert(not err:find("^.-:%d+: "), "rawset nil has file:line prefix: " .. err)
assert(not err2:find("^.-:%d+: "), "rawset NaN has file:line prefix: " .. err2)
assert(not err3:find("^.-:%d+: "), "unpack nil has file:line prefix: " .. err3)
assert(not err4:find("^.-:%d+: "), "sort has file:line prefix: " .. err4)

print("OK")
