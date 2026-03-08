-- Test: load/dump - Binary chunk round-tripping
-- From: db.lua, literals.lua
-- What: Tests string.dump and load for binary chunk serialization

-- Basic round-trip
do
  local f = function() return 42 end
  local dumped = string.dump(f)
  assert(type(dumped) == "string")
  local g, err = load(dumped)
  assert(g, "load(string.dump(f)) should succeed, got: " .. tostring(err))
  assert(g() == 42)
end

-- Round-trip with arguments
do
  local f = function(x) return x * 2 + 1 end
  local dumped = string.dump(f)
  local g = assert(load(dumped))
  assert(g(10) == 21)
  assert(g(0) == 1)
  assert(g(-5) == -9)
end

-- Multiple return values
do
  local f = function() return 1, 2, 3 end
  local dumped = string.dump(f)
  local g = assert(load(dumped))
  local a, b, c = g()
  assert(a == 1 and b == 2 and c == 3)
end

-- Stripped dump (no debug info)
do
  local f = function(x) return x * 2 + 1 end
  local full = string.dump(f)
  local stripped = string.dump(f, true)
  assert(#stripped <= #full, "stripped dump should not be larger")
  local g = assert(load(stripped))
  assert(g(10) == 21)
end

-- Various value types survive round-trip
do
  local f = function()
    return nil, true, false, 0, 1.5, "str", 2^53
  end
  local g = assert(load(string.dump(f)))
  local a, b, c, d, e, s, big = g()
  assert(a == nil)
  assert(b == true)
  assert(c == false)
  assert(d == 0)
  assert(e == 1.5)
  assert(s == "str")
  assert(big == 2^53)
end

-- Empty function
do
  local f = function() end
  local g = assert(load(string.dump(f)))
  assert(g() == nil)
end

-- Binary chunk with mode "t" rejected
do
  local f = function() return 99 end
  local bin = string.dump(f)
  local g, err = load(bin, "test", "t")
  assert(g == nil, "should reject binary chunk in text mode")
  assert(err:find("binary"), "error should mention 'binary', got: " .. tostring(err))
end

-- Binary chunk with mode "b" accepted
do
  local f = function(x) return x + 1 end
  local bin = string.dump(f)
  local g = assert(load(bin, "test", "b"))
  assert(g(5) == 6)
end

-- Binary chunk with mode "bt" accepted
do
  local f = function() return "hello" end
  local bin = string.dump(f)
  local g = assert(load(bin, "test", "bt"))
  assert(g() == "hello")
end

-- Text chunk with mode "b" rejected
do
  local g, err = load("return 5", "test", "b")
  assert(g == nil, "binary mode should reject text chunks")
  assert(type(err) == "string")
end

-- Shebang NOT stripped from string args to load()
do
  local s = "#!/usr/bin/lua\nreturn 43"
  local f, err = load(s)
  assert(f == nil, "load() should NOT strip shebang from string args")
  assert(err:find("unexpected symbol") or err:find("#"),
    "error should mention unexpected symbol, got: " .. tostring(err))
end

print("OK")
