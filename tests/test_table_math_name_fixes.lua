-- Tests for __name in error messages, table.move validation, and tableObjLen prefix

-- Issue 1: __name should be used in length operator error messages
-- Use io.stdout which is a userdata with __name = "FILE*"
do
  local ok, err = pcall(function() return #io.stdout end)
  assert(not ok, "expected error")
  assert(err:find("FILE*", 1, true), "length error should use __name, got: " .. err)
end

-- Issue 2: __name should be used in table function error messages (tableCheckLike)
do
  -- We need a real userdata with __name to test this properly
  -- io.stdout has __name = "FILE*"
  local ok, err = pcall(table.insert, io.stdout, 1)
  assert(not ok, "expected error")
  assert(err:find("FILE*", 1, true), "table.insert error should use __name, got: " .. err)
end

-- Issue 3: table.move without 5th arg requires table-like (read+write) for arg 1
do
  local ok, err = pcall(table.move, "hello", 1, 1, 1)
  assert(not ok, "expected error for string arg to table.move")
  assert(err:find("table expected, got string"), "table.move should reject string, got: " .. err)
end

-- Issue 3: table.move with empty range should still validate (no 5th arg)
do
  local ok, err = pcall(table.move, "hello", 2, 1, 1)
  assert(not ok, "expected error for string arg to table.move even with empty range")
  assert(err:find("table expected, got string"), "table.move empty range should reject string, got: " .. err)
end

-- Issue 3: table.move with 5th arg validates arg 5 needs __newindex
do
  local ok, err = pcall(table.move, {}, 1, 2, 3, "bar")
  assert(not ok, "expected error for string destination in table.move")
  assert(err:find("table expected, got string"), "table.move should reject string dest, got: " .. err)
end

-- Issue 3: __name should be used in table.move error message
do
  local ok, err = pcall(table.move, io.stdout, 1, 1, 1)
  assert(not ok, "expected error for FILE* arg to table.move")
  assert(err:find("FILE*", 1, true), "table.move should use __name, got: " .. err)
end

-- Issue 4: "object length is not an integer" should have file:line prefix
do
  local t = setmetatable({}, {__len = function() return "hello" end})
  local ok, err = pcall(function() table.insert(t, "x") end)
  assert(not ok, "expected error")
  -- Error should contain file:line prefix (the file name from this script)
  assert(err:find(":%d+:"), "error should have file:line prefix, got: " .. err)
  assert(err:find("object length is not an integer"), "wrong error message, got: " .. err)
end

print("PASS")
