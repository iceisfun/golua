-- Differential testing round 4: behavioral bug fixes
-- Found via automated differential testing against Lua 5.4.8

-- Bug 1: xpcall error leak after pcall
-- pcall inside xpcall must not leak caught errors to xpcall's handler
do
  local ok, err = xpcall(function()
    pcall(error, "INNER")
    error("OUTER")
  end, function(e) return e end)
  assert(not ok)
  assert(type(err) == "string" and err:find("OUTER"), "xpcall leak: " .. tostring(err))

  -- Multiple pcalls before error
  local ok2, err2 = xpcall(function()
    pcall(error, "A")
    pcall(error, "B")
    pcall(error, "C")
    error("FINAL")
  end, function(e) return e end)
  assert(not ok2)
  assert(err2:find("FINAL"), "multi pcall leak: " .. tostring(err2))

  -- Nested xpcall inside pcall inside xpcall
  local ok3, err3 = xpcall(function()
    pcall(function()
      xpcall(function() error("DEEP") end, function(e) return "h:" .. e end)
    end)
    error("SHALLOW")
  end, function(e) return e end)
  assert(not ok3)
  assert(err3:find("SHALLOW"), "nested leak: " .. tostring(err3))

  -- pcall success then error in xpcall
  local ok4, err4 = xpcall(function()
    pcall(function() return 42 end)
    error("AFTER")
  end, function(e) return e end)
  assert(not ok4)
  assert(err4:find("AFTER"))
end

-- Bug 4: pairs(nil) / ipairs(nil) should not error
do
  local f, s, v = pairs(nil)
  assert(f == next, "pairs(nil) should return next")
  assert(s == nil)
  assert(v == nil)

  local f2, s2, v2 = ipairs(nil)
  assert(type(f2) == "function")
  assert(s2 == nil)
  assert(v2 == 0)

  -- pairs/ipairs with no args should still error
  local ok1, err1 = pcall(pairs)
  assert(not ok1 and err1:find("value expected"))
  local ok2, err2 = pcall(ipairs)
  assert(not ok2 and err2:find("value expected"))
end

-- Bug 5: pairs() returns same next as global
do
  local f = pairs({})
  assert(f == next, "pairs({}) should return global next")
  local f2 = pairs({1,2,3})
  assert(f2 == next)
end

-- Bug 10: warn(42) should coerce number to string
do
  local ok, err = pcall(warn, 42)
  assert(ok, "warn(42) should succeed: " .. tostring(err))
  local ok2, err2 = pcall(warn, 3.14)
  assert(ok2, "warn(3.14) should succeed: " .. tostring(err2))
  -- Non-string/number should still error
  local ok3, _ = pcall(warn, true)
  assert(not ok3, "warn(true) should error")
  local ok4, _ = pcall(warn, {})
  assert(not ok4, "warn({}) should error")
end

-- Bug 11: collectgarbage(42) should error
do
  local ok, err = pcall(collectgarbage, 42)
  assert(not ok, "collectgarbage(42) should error")
  assert(err:find("invalid option"), err)
  -- Valid string options should still work
  local ok2, _ = pcall(collectgarbage, "collect")
  assert(ok2)
  local ok3, _ = pcall(collectgarbage, "count")
  assert(ok3)
end

print("PASS")
