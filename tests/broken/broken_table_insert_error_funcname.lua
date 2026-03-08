-- table.insert wrong number of arguments error should use bare name 'insert'
-- when called from Lua code (not 'table.insert')
local ok, err

ok, err = pcall(function() table.insert({}) end)
assert(err:find("'insert'") and not err:find("'table.insert'"),
  "expected bare name 'insert' in wrong-args error, got: " .. tostring(err))

ok, err = pcall(function() table.insert({}, 1, 2, 3) end)
assert(err:find("'insert'") and not err:find("'table.insert'"),
  "expected bare name 'insert' in wrong-args error, got: " .. tostring(err))

print("OK")
