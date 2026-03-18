-- Weak table tests matching Lua 5.4 __mode semantics.

-- Weak values: collectable values are removed after GC.
do
  local t = setmetatable({}, {__mode = "v"})
  local obj = {}
  t.key = obj
  assert(t.key == obj, "weak value: should be accessible while alive")

  obj = nil
  collectgarbage()

  assert(t.key == nil, "weak value: should be nil after collection")
end

-- Weak keys: collectable keys are removed after GC.
do
  local t = setmetatable({}, {__mode = "k"})
  local key = {}
  t[key] = "value"
  assert(t[key] == "value", "weak key: should be accessible while alive")

  key = nil
  collectgarbage()

  -- Verify entry was removed via pairs.
  local count = 0
  for _ in pairs(t) do count = count + 1 end
  assert(count == 0, "weak key: entry should be gone after collection, got " .. count)
end

-- Weak kv: both keys and values are weak.
do
  local t = setmetatable({}, {__mode = "kv"})
  local key = {}
  local val = {}
  t[key] = val

  -- Drop value only.
  val = nil
  collectgarbage()

  local count = 0
  for _ in pairs(t) do count = count + 1 end
  assert(count == 0, "weak kv: entry should be gone when value is collected")
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
  t["permanent"] = "value"
  t[42] = "number_key"

  collectgarbage()

  assert(t["permanent"] == "value", "string key should survive")
  assert(t[42] == "number_key", "number key should survive")
end

-- pairs() skips dead entries.
do
  local t = setmetatable({}, {__mode = "v"})
  local alive = {}
  t.alive = alive
  t.dead = {}
  t.also_alive = 123

  collectgarbage()

  local found = {}
  for k in pairs(t) do found[k] = true end

  assert(found.alive, "alive entry should appear in pairs")
  assert(not found.dead, "dead entry should not appear in pairs")
  assert(found.also_alive, "value-type entry should appear in pairs")

  -- Keep alive reference to prevent premature collection.
  local _ = alive
end

-- # operator works on weak tables.
do
  local t = setmetatable({}, {__mode = "v"})
  t[1] = "one"
  t[2] = "two"
  t[3] = "three"

  assert(#t == 3, "length should be 3, got " .. #t)
end

-- Transition: setting metatable with __mode migrates entries.
do
  local t = {}
  t.key = "value"
  t[1] = "one"

  setmetatable(t, {__mode = "v"})

  assert(t.key == "value", "data should survive weak mode transition")
  assert(t[1] == "one", "array data should survive weak mode transition")

  -- Transition back to strong.
  setmetatable(t, {})

  assert(t.key == "value", "data should survive strong mode transition")
  assert(t[1] == "one", "array data should survive strong mode transition")
end

-- Function values as weak values are collected.
do
  local t = setmetatable({}, {__mode = "v"})
  t.f = function() return 1 end

  collectgarbage()

  assert(t.f == nil, "function should be collected from weak-value table")
end

-- Function values as weak keys are collected.
do
  local t = setmetatable({}, {__mode = "k"})
  local f = function() return 1 end
  t[f] = "data"
  assert(t[f] == "data")

  f = nil
  collectgarbage()

  local count = 0
  for _ in pairs(t) do count = count + 1 end
  assert(count == 0, "function key should be collected")
end

print("OK")
