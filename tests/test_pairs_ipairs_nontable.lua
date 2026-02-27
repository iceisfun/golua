-- Test: pairs() and ipairs() should accept non-table values
-- Lua 5.4 does not eagerly validate the type of the first argument.
-- It returns the iterator triple, and the error occurs when the iterator
-- is first invoked (trying to index the non-table). GoLua rejects
-- non-tables too early at call time.

-- pairs(42) should not error; it returns (next, 42, nil).
-- Only iterating should fail.
local ok1, iter, state, init = pcall(pairs, 42)
assert(ok1, "pairs(42) should not error at call time, got: " .. tostring(iter))

-- But iterating a number should fail
local ok2, err2 = pcall(function()
  for k, v in pairs(42) do end
end)
assert(not ok2, "pairs(42) iteration should error")

-- ipairs(42) should not error at call time either
local ok3, iter3, state3, init3 = pcall(ipairs, 42)
assert(ok3, "ipairs(42) should not error at call time, got: " .. tostring(iter3))

-- But iterating a number should fail
local ok4, err4 = pcall(function()
  for i, v in ipairs(42) do end
end)
assert(not ok4, "ipairs(42) iteration should error")
