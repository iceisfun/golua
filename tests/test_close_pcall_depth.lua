-- coroutine.close + __close handler pcall depth semantics (Lua 5.5).
--
-- Regression for the closingCoroutine over-broad re-panic: a pcall created
-- *inside* a __close handler must catch its own errors normally, while an
-- uncaught __close error must still escape any *suspended* user pcall and
-- reach coroutine.close.

-- Case 1: pcall inside the handler catches normally; close succeeds.
do
  local co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function()
      local ok, err = pcall(function() error("e") end)
      assert(ok == false, "inner pcall should return false")
      assert(tostring(err):find("e"), "inner err preserved")
    end})
    coroutine.yield()
  end)
  coroutine.resume(co)
  local ok = coroutine.close(co)
  assert(ok == true, "close should succeed when handler catches its own error")
end

-- Case 2: uncaught handler error escapes a SUSPENDED user pcall and reaches
-- coroutine.close (the suspended pcall must NOT swallow it).
do
  local reached = false
  local co = coroutine.create(function()
    pcall(function()
      local x <close> = setmetatable({}, {__close = function() error("boom") end})
      coroutine.yield()
    end)
    reached = true
  end)
  coroutine.resume(co)
  local ok, err = coroutine.close(co)
  assert(ok == false, "close should report the uncaught handler error")
  assert(tostring(err):find("boom"), "error value should propagate")
  assert(reached == false, "code after the suspended pcall must not run")
end

-- Case 3: handler with both an internal caught pcall AND a suspended outer
-- pcall; the internal error is caught, close still succeeds.
do
  local co = coroutine.create(function()
    pcall(function()
      local x <close> = setmetatable({}, {__close = function()
        local ok = pcall(function() error("inner") end)
        assert(ok == false, "handler-internal pcall catches")
      end})
      coroutine.yield()
    end)
  end)
  coroutine.resume(co)
  local ok = coroutine.close(co)
  assert(ok == true, "close succeeds; internal handler error stays contained")
end

print("OK")
--> =OK
