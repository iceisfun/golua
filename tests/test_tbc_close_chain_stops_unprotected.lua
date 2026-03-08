-- Test: __close chain stops after error in unprotected coroutine context
-- In Lua 5.4, when a __close metamethod errors during unprotected scope exit
-- (e.g., coroutine return), the chain stops — earlier TBC vars do NOT get closed.
-- In pcall context, the chain continues (all handlers run).

-- Case 1: Coroutine resume (unprotected) — chain should STOP
do
  local log = {}
  local co = coroutine.create(function()
    local a <close> = setmetatable({}, {__close=function(_, err)
      log[#log+1] = "a"
    end})
    local b <close> = setmetatable({}, {__close=function()
      error("b close error")
    end})
    return "success"
  end)
  local ok, err = coroutine.resume(co)
  assert(not ok, "resume should fail")
  assert(string.find(err, "b close error"), "error should mention b: " .. tostring(err))
  -- Key assertion: a's __close should NOT have been called
  assert(#log == 0, "expected 0 close calls (chain should stop), got " .. #log)
end

-- Case 2: pcall context — chain should CONTINUE (all handlers run)
do
  local log = {}
  local ok, err = pcall(function()
    local a <close> = setmetatable({}, {__close=function(_, err)
      log[#log+1] = "a"
    end})
    local b <close> = setmetatable({}, {__close=function()
      error("b close error")
    end})
    return "success"
  end)
  assert(not ok, "pcall should fail")
  assert(string.find(err, "b close error"), "error should mention b: " .. tostring(err))
  -- In pcall context, all handlers run
  assert(#log == 1, "expected 1 close call in pcall context, got " .. #log)
end

-- Case 3: Three TBC vars, middle one errors — only later ones should close
do
  local log = {}
  local co = coroutine.create(function()
    local a <close> = setmetatable({}, {__close=function(_, err)
      log[#log+1] = "a"
    end})
    local b <close> = setmetatable({}, {__close=function()
      error("b close error")
    end})
    local c <close> = setmetatable({}, {__close=function(_, err)
      log[#log+1] = "c"
    end})
    return "done"
  end)
  local ok, err = coroutine.resume(co)
  assert(not ok)
  -- c closes first (no error), b errors, a should NOT close
  assert(#log == 1, "expected 1 close call (only c), got " .. #log)
  assert(log[1] == "c", "expected c to close, got " .. tostring(log[1]))
end

print("PASS")
