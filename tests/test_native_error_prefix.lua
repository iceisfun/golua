-- Test that errors from native/C functions don't have spurious file:line prefix

-- rawset with nil key
local ok, err = pcall(rawset, {}, nil, 1)
assert(not ok)
assert(err == "table index is nil", "rawset nil: expected 'table index is nil', got: " .. tostring(err))

-- rawset with NaN key
ok, err = pcall(rawset, {}, 0/0, 1)
assert(not ok)
assert(err == "table index is NaN", "rawset NaN: expected 'table index is NaN', got: " .. tostring(err))

-- table.unpack with nil
ok, err = pcall(table.unpack, nil)
assert(not ok)
assert(err == "attempt to get length of a nil value", "unpack nil: got: " .. tostring(err))

-- table.unpack with number
ok, err = pcall(table.unpack, 42)
assert(not ok)
assert(err == "attempt to get length of a number value", "unpack num: got: " .. tostring(err))

-- coroutine.yield from main thread
ok, err = pcall(coroutine.yield)
assert(not ok)
assert(err == "attempt to yield from outside a coroutine", "yield: got: " .. tostring(err))

-- table.sort comparison error
ok, err = pcall(table.sort, {1, "a"})
assert(not ok)
assert(not err:find("^[^:]+:%d+:"), "table.sort error should not have file:line prefix: " .. tostring(err))

print("OK")
