-- load(reader [, chunkname [, mode [, env]]]) compatibility checks.

local function assertErrContains(err, s)
  assert(type(err) == "string", "error should be string, got " .. type(err))
  assert(err:find(s, 1, true), "expected error containing '" .. s .. "', got: " .. tostring(err))
end

-- Reader returning empty string ends input immediately.
do
  local calls = 0
  local f, err = load(function()
    calls = calls + 1
    if calls == 1 then return "" end
    error("reader should not be called after empty chunk")
  end)
  assert(f ~= nil, "empty first chunk should produce empty chunk function: " .. tostring(err))
  assert(f() == nil)
  assert(calls == 1)
end

-- Reader returning no values behaves like nil (end of input).
do
  local f, err = load(function() end)
  assert(f ~= nil, "reader with no return values should succeed: " .. tostring(err))
  assert(f() == nil)
end

-- Reader numbers are accepted and coerced to strings.
do
  local i = 0
  local f, err = load(function()
    i = i + 1
    if i == 1 then return "return " end
    if i == 2 then return 123 end
    return nil
  end)
  assert(f ~= nil, "numeric reader fragments should compile: " .. tostring(err))
  assert(f() == 123)
end

-- Reader invalid type should fail with the canonical message.
do
  local f, err = load(function() return true end)
  assert(f == nil)
  assertErrContains(err, "reader function must return a string")
end

-- Reader errors: direct calls stringify non-string error objects,
-- while pcall(load, ...) preserves them.
do
  local f1, e1 = load(function() error("boom") end)
  assert(f1 == nil)
  assert(type(e1) == "string" and e1:find("boom", 1, true), "unexpected string error: " .. tostring(e1))

  local f2, e2 = load(function() error({ code = 42 }) end)
  assert(f2 == nil)
  assert(type(e2) == "string")
  assertErrContains(e2, "error object is a table value")

  local ok, f3, e3 = pcall(load, function() error({ code = 99 }) end)
  assert(ok)
  assert(f3 == nil)
  assert(type(e3) == "table" and e3.code == 99)
end

-- Mode handling for text/binary and non-canonical mode strings.
do
  local done = false
  local f1, e1 = load(function()
    if done then return nil end
    done = true
    return "return 1"
  end, "x", "b")
  assert(f1 == nil)
  assertErrContains(e1, "attempt to load a text chunk")

  local f2 = assert(load("return 4", "x", "tt"))
  assert(f2() == 4)

  local f3 = assert(load("return 5", "x", "tbx"))
  assert(f3() == 5)

  local dumped = string.dump(function() return 9 end)
  local f4, e4 = load(dumped, "x", "t")
  assert(f4 == nil)
  assertErrContains(e4, "attempt to load a binary chunk")
end

-- Explicit env (including nil) is respected for reader chunks.
do
  local env = { x = 77 }
  local done1 = false
  local f1 = assert(load(function()
    if done1 then return nil end
    done1 = true
    return "return x"
  end, "x", "t", env))
  assert(f1() == 77)

  local done2 = false
  local f2 = assert(load(function()
    if done2 then return nil end
    done2 = true
    return "return _ENV"
  end, "x", "t", nil))
  assert(f2() == nil)
end

-- Deterministic stateful reader with crafted pseudo-random expression.
do
  math.randomseed(123456)
  local parts = { "return " }
  local sum = 0
  for i = 1, 8 do
    local n = math.random(0, 9)
    sum = sum + n
    parts[#parts + 1] = tostring(n)
    if i < 8 then
      parts[#parts + 1] = "+"
    end
  end

  local idx = 0
  local f, err = load(function()
    idx = idx + 1
    return parts[idx]
  end)
  assert(f ~= nil, "stateful reader should compile: " .. tostring(err))
  assert(f() == sum)
end
