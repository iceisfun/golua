-- When closing a coroutine with multiple <close> variables, if one __close
-- handler errors, the error should be passed as the second argument to the
-- next __close handler in the chain.
--
-- Lua 5.4 Reference §3.3.8: "If a closing method raises an error, that error
-- is handled like an error in the regular code where the variable was defined."
-- The error becomes the "original error" passed to subsequent __close handlers.

-- Test 1: error propagation during coroutine.close()
local log = {}
local co = coroutine.create(function()
  local a <close> = setmetatable({}, {__close = function(self, err)
    log[#log+1] = "a:" .. tostring(err)
  end})
  local b <close> = setmetatable({}, {__close = function(self, err)
    log[#log+1] = "b:" .. tostring(err)
    error("b-fail")
  end})
  coroutine.yield()
end)

coroutine.resume(co)
local ok, err = coroutine.close(co)

-- b closes first (reverse order), receives nil (no prior error)
assert(log[1] == "b:nil", "b should receive nil, got: " .. tostring(log[1]))
-- a closes second, should receive b's error
assert(log[2] ~= nil, "a's __close should have been called")
assert(string.find(log[2], "b%-fail"),
  "a should receive b's error, got: " .. tostring(log[2]))

-- Test 2: error propagation in normal scope exit (non-coroutine)
local log2 = {}
local ok2, err2 = pcall(function()
  local x <close> = setmetatable({}, {__close = function(self, err)
    log2[#log2+1] = "x:" .. tostring(err)
  end})
  local y <close> = setmetatable({}, {__close = function(self, err)
    log2[#log2+1] = "y:" .. tostring(err)
    error("y-fail")
  end})
  error("original")
end)

assert(log2[1] ~= nil and string.find(log2[1], "y:.*original"),
  "y should receive 'original' error, got: " .. tostring(log2[1]))
assert(log2[2] ~= nil and string.find(log2[2], "x:.*y%-fail"),
  "x should receive y's error (which replaced original), got: " .. tostring(log2[2]))

print("PASS")
