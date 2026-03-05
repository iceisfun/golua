-- ==========================================================================
-- Fengari test extraction: Debug internals (error message format)
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: ldebug
-- Total tests: 7
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Tests verify that type errors produce appropriate error messages.
-- Original tests expected errors to be shown; fixed to use pcall.

-- [Test 1] luaG_typeerror: # on boolean
do
  local a = true
  local ok, err = pcall(function() return #a end)
  assert(not ok and string.find(err, "attempt to get length of a boolean"))
  print("PASS")
end
--> =PASS

-- [Test 2] luaG_typeerror: index on boolean
do
  local a = true
  local ok, err = pcall(function() return a.yo end)
  assert(not ok and string.find(err, "attempt to index a boolean"))
  print("PASS")
end
--> =PASS

-- [Test 3] luaG_typeerror: index on boolean (duplicate of test 2)
do
  local a = true
  local ok, err = pcall(function() return a.yo end)
  assert(not ok and string.find(err, "attempt to index a boolean"))
  print("PASS")
end
--> =PASS

-- [Test 4] luaG_typeerror: newindex on boolean
do
  local a = true
  local ok, err = pcall(function() a.yo = 1 end)
  assert(not ok and string.find(err, "attempt to index a boolean"))
  print("PASS")
end
--> =PASS

-- [Test 5] luaG_concaterror: concat table with string
do
  local ok, err = pcall(function() return {} .. 'hello' end)
  assert(not ok and string.find(err, "attempt to concatenate"))
  print("PASS")
end
--> =PASS

-- [Test 6] luaG_opinterror: add table with string
do
  local ok, err = pcall(function() return {} + 'hello' end)
  assert(not ok and string.find(err, "attempt to perform arithmetic"))
  print("PASS")
end
--> =PASS

-- [Test 7] luaG_tointerror: bitwise on float
do
  local ok, err = pcall(function() return 123.5 & 12 end)
  assert(not ok and string.find(err, "number has no integer representation"))
  print("PASS")
end
--> =PASS
