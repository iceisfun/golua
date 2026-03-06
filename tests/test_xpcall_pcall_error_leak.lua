-- xpcall must not leak pcall'd errors into its message handler
-- Bug: when pcall catches an error inside xpcall, the next error
-- delivered the old pcall'd error to the xpcall handler instead.

-- Basic case: pcall inside xpcall, then error
local ok, err = xpcall(function()
  pcall(error, "INNER")
  error("OUTER")
end, function(e) return e end)
assert(not ok)
assert(type(err) == "string" and err:find("OUTER"), "expected OUTER, got: " .. tostring(err))

-- Multiple pcalls before the error
local ok2, err2 = xpcall(function()
  pcall(error, "A")
  pcall(error, "B")
  pcall(error, "C")
  error("FINAL")
end, function(e) return e end)
assert(not ok2)
assert(type(err2) == "string" and err2:find("FINAL"), "expected FINAL, got: " .. tostring(err2))

-- pcall success followed by error
local ok3, err3 = xpcall(function()
  pcall(function() return 42 end)
  error("AFTER_SUCCESS")
end, function(e) return e end)
assert(not ok3)
assert(type(err3) == "string" and err3:find("AFTER_SUCCESS"), "expected AFTER_SUCCESS, got: " .. tostring(err3))

-- Nested xpcall inside pcall inside xpcall
local ok4, err4 = xpcall(function()
  pcall(function()
    xpcall(function()
      error("DEEP")
    end, function(e) return "handled:" .. e end)
  end)
  error("SHALLOW")
end, function(e) return e end)
assert(not ok4)
assert(type(err4) == "string" and err4:find("SHALLOW"), "expected SHALLOW, got: " .. tostring(err4))

print("PASS")
