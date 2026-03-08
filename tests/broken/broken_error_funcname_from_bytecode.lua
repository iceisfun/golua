-- When a native/stdlib function is called from Lua code, error messages should
-- use the name resolved from bytecode debug info (short name), not the
-- registered native function name (qualified name).
-- e.g., math.abs("x") from Lua code should say "'abs'" not "'math.abs'"
local ok, err

-- math.abs called directly in Lua code
ok, err = pcall(function() math.abs("hello") end)
-- Lua 5.4 resolves the name from bytecode: 'abs'
assert(err:find("'abs'") and not err:find("'math.abs'"),
  "expected short name 'abs', got: " .. tostring(err))

-- string.format called directly in Lua code
ok, err = pcall(function() string.format("%d", "hello") end)
assert(err:find("'format'") and not err:find("'string.format'"),
  "expected short name 'format', got: " .. tostring(err))

-- coroutine.resume called directly in Lua code
ok, err = pcall(function() coroutine.resume("bad") end)
assert(err:find("'resume'") and not err:find("'coroutine.resume'"),
  "expected short name 'resume', got: " .. tostring(err))

-- table.insert with wrong args from Lua code
ok, err = pcall(function() table.insert({}) end)
assert(err:find("'insert'") and not err:find("'table.insert'"),
  "expected short name 'insert', got: " .. tostring(err))

-- When renamed via local, should use the local's name
ok, err = pcall(function() local myabs = math.abs; myabs("hello") end)
assert(err:find("'myabs'"),
  "expected local name 'myabs', got: " .. tostring(err))

print("OK")
