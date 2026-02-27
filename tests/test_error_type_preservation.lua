-- Bug: error values (tables) are converted to strings when propagating
-- through native function callbacks (table.sort, string.gsub, etc.)
-- due to panic(err.Error()) instead of re-panicking the *vm.LuaError.

-- Test 1: error in table.sort comparator preserves table error
local ok1, err1 = pcall(table.sort, {3, 1, 2}, function(a, b)
  error({msg = "sort error"})
end)
assert(not ok1, "pcall should fail")
assert(type(err1) == "table", "error should be table, got " .. type(err1))
assert(err1.msg == "sort error", "error.msg should be 'sort error'")

-- Test 2: error in string.gsub function replacement preserves table error
local ok2, err2 = pcall(string.gsub, "abc", ".", function()
  error({msg = "gsub error"})
end)
assert(not ok2, "pcall should fail")
assert(type(err2) == "table", "error should be table, got " .. type(err2))
assert(err2.msg == "gsub error", "error.msg should be 'gsub error'")

-- Test 3: error in table.sort with integer error value
local ok3, err3 = pcall(table.sort, {2, 1}, function(a, b)
  error(42)
end)
assert(not ok3, "pcall should fail")
assert(type(err3) == "number", "error should be number, got " .. type(err3))
assert(err3 == 42, "error should be 42, got " .. tostring(err3))

print("PASS")
