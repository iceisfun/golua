-- coroutine.isyieldable() argument validation
-- Bug: GoLua accepted nil/non-thread args instead of erroring

-- No args: check current thread (valid)
assert(coroutine.isyieldable() == false)

-- Valid thread arg
local co = coroutine.create(function() coroutine.yield() end)
assert(coroutine.isyieldable(co) == true)

-- nil arg: should error
local ok, err = pcall(coroutine.isyieldable, nil)
assert(not ok)
assert(err:find("thread expected, got nil"), "nil: " .. tostring(err))

-- number arg: should error
local ok2, err2 = pcall(coroutine.isyieldable, 42)
assert(not ok2)
assert(err2:find("thread expected, got number"), "number: " .. tostring(err2))

-- string arg: should error
local ok3, err3 = pcall(coroutine.isyieldable, "hello")
assert(not ok3)
assert(err3:find("thread expected, got string"), "string: " .. tostring(err3))

-- table (non-thread) arg: should error
local ok4, err4 = pcall(coroutine.isyieldable, {})
assert(not ok4)
assert(err4:find("thread expected, got table"), "table: " .. tostring(err4))

print("PASS")
