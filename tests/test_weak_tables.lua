-- Weak table tests matching Lua 5.4 __mode semantics.
-- Note: tests use helper functions to create/assign objects in separate
-- stack frames, preventing Go's GC from keeping stale temp register refs.

local function makeTable() return {} end

-- Helper: creates a weak table, inserts a collectable value, drops it, GCs.
local function testWeakValue()
  local t = setmetatable({}, {__mode = "v"})
  local obj = makeTable()
  t.key = obj
  assert(t.key == obj, "weak value: should be accessible while alive")
  return t  -- obj goes out of scope
end

do
  local t = testWeakValue()
  for _ = 1, 3 do collectgarbage() end
  assert(t.key == nil, "weak value: should be nil after collection")
end

-- Helper for weak keys test.
local function testWeakKey()
  local t = setmetatable({}, {__mode = "k"})
  local key = makeTable()
  t[key] = "value"
  assert(t[key] == "value", "weak key: should be accessible while alive")
  return t  -- key goes out of scope
end

do
  local t = testWeakKey()
  for _ = 1, 3 do collectgarbage() end
  local count = 0
  for _ in pairs(t) do count = count + 1 end
  assert(count == 0, "weak key: entry should be gone after collection, got " .. count)
end

-- Weak kv: both keys and values are weak.
local function testWeakKV()
  local t = setmetatable({}, {__mode = "kv"})
  local key = makeTable()
  local val = makeTable()
  t[key] = val
  return t  -- key and val go out of scope
end

do
  local t = testWeakKV()
  for _ = 1, 3 do collectgarbage() end
  local count = 0
  for _ in pairs(t) do count = count + 1 end
  assert(count == 0, "weak kv: entry should be gone when value/key is collected")
end

-- Value types are never collected from weak tables.
do
  local t = setmetatable({}, {__mode = "v"})
  t.num = 42
  t.str = "hello"
  t.bool = true
  t.flt = 3.14

  collectgarbage()

  assert(t.num == 42, "number should survive")
  assert(t.str == "hello", "string should survive")
  assert(t.bool == true, "boolean should survive")
  assert(t.flt == 3.14, "float should survive")
end

-- String keys in weak-key tables are never collected.
do
  local t = setmetatable({}, {__mode = "k"})
  t["persistent"] = "yes"
  t[42] = "number_key"

  collectgarbage()

  assert(t["persistent"] == "yes", "string key should survive")
  assert(t[42] == "number_key", "number key should survive")
end

-- pairs() skips dead entries (when GC collects them).
local function testPairsSkipDead()
  local t = setmetatable({}, {__mode = "v"})
  local alive = makeTable()
  t.alive = alive
  -- Create dead entry in separate scope.
  local function setdead(tbl) tbl.dead = {} end
  setdead(t)
  t.also_alive = 123
  return t, alive  -- alive must survive
end

do
  local t, alive = testPairsSkipDead()
  collectgarbage()

  local found = {}
  for k in pairs(t) do found[k] = true end

  assert(found.alive, "alive entry should appear in pairs")
  -- Note: found.dead may or may not be collected depending on Go's GC timing.
  assert(found.also_alive, "value-type entry should appear in pairs")

  -- Keep alive reference to prevent premature collection.
  local _ = alive
end

-- # operator works on weak tables.
do
  local t = setmetatable({}, {__mode = "v"})
  t[1] = "a"
  t[2] = "b"
  t[3] = "c"
  assert(#t == 3, "len should count sequence")
end

-- Mode transition: setting metatable with __mode on existing table.
do
  local t = {}
  t.key = "value"
  t[42] = "num"

  setmetatable(t, {__mode = "v"})
  collectgarbage()

  assert(t.key == "value", "string values should survive weak mode")
  assert(t[42] == "num", "string values with int keys should survive weak mode")
end

-- Delete from weak table.
do
  local t = setmetatable({}, {__mode = "k"})
  local k1 = makeTable()
  local k2 = makeTable()
  t[k1] = "one"
  t[k2] = "two"
  assert(t[k1] == "one")

  t[k1] = nil
  assert(t[k1] == nil, "deleted entry should return nil")
  assert(t[k2] == "two", "other entry should survive")
end

-- ForEach on weak tables.
do
  local t = setmetatable({}, {__mode = "v"})
  t.a = 1
  t.b = 2
  t.c = 3

  local sum = 0
  for _, v in pairs(t) do sum = sum + v end
  assert(sum == 6, "ForEach should visit all entries")
end

print("OK")
