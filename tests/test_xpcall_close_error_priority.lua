-- When both a function error and a __close error occur, xpcall's handler
-- should receive the __close error (not the original function error).
-- In Lua 5.4, __close errors replace the original error.
local ok, err

-- Both function and __close error: handler should see __close error
ok, err = xpcall(function()
  local x <close> = setmetatable({}, {__close = function()
    error("CLOSE")
  end})
  error("MAIN")
end, function(e) return tostring(e) end)
assert(not ok)
assert(err:find("CLOSE"), "expected __close error 'CLOSE', got: " .. tostring(err))
assert(not err:find("MAIN"), "should not contain main error 'MAIN', got: " .. tostring(err))

-- Non-string __close error replaces string main error
ok, err = xpcall(function()
  local x <close> = setmetatable({}, {__close = function()
    error(42)
  end})
  error("MAIN")
end, function(e) return type(e) .. ":" .. tostring(e) end)
assert(not ok)
assert(err == "number:42", "expected 'number:42', got: " .. tostring(err))

-- String __close error replaces non-string main error
ok, err = xpcall(function()
  local x <close> = setmetatable({}, {__close = function()
    error("CLOSE")
  end})
  error({msg="MAIN"})
end, function(e) return type(e) .. ":" .. tostring(e) end)
assert(not ok)
assert(err:find("string:"), "expected string __close error, got: " .. tostring(err))
assert(err:find("CLOSE"), "expected 'CLOSE' in error, got: " .. tostring(err))

print("OK")
