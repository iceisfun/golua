-- Test: load() reader byte-size and call-count limits
-- Verifies that load() with a reader function that returns data indefinitely
-- does not hang but instead errors when limits are exceeded.

-- Test 1: Reader that returns nil after many iterations (should work fine)
do
  local n = 0
  local ok, err = load(function()
    n = n + 1
    if n > 1000 then return nil end
    return "local x=1 "
  end)
  -- This should either compile successfully or return a parse error,
  -- but must NOT hang.
  assert(ok ~= nil or err ~= nil, "load must return something, not hang")
end

-- Test 2: Reader that returns tiny fragments indefinitely hits call limit
do
  local ok, err = load(function()
    return "1+"  -- 2 bytes each, never returns nil
  end)
  -- Must not hang; should return nil + error.
  -- May report a syntax error (early detection) or "not enough memory" (limit hit).
  assert(ok == nil, "infinite reader must fail, not hang")
  assert(type(err) == "string", "error must be a string")
  assert(string.find(err, "not enough memory") or string.find(err, "unexpected symbol"),
    "expected 'not enough memory' or syntax error, got: " .. tostring(err))
end

-- Test 3: Reader that returns larger chunks hits byte limit
do
  local big = string.rep("a", 1024 * 1024) -- 1 MB of 'a' per call
  local ok, err = load(function()
    return big  -- 1MB each call, never stops
  end)
  assert(ok == nil, "infinite large-chunk reader must fail")
  assert(type(err) == "string", "error must be a string")
end

-- Test 4: Reader that returns "local x=1 " indefinitely (original bug report)
do
  local ok, err = load(function()
    return "local x=1 "  -- never returns nil
  end)
  assert(ok == nil, "infinite 'local x=1 ' reader must fail")
  assert(type(err) == "string", "error must be a string")
end

print("PASSED: load() reader limit tests")
