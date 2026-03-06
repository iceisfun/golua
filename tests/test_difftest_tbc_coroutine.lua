-- Differential testing: TBC variable closing in coroutines
-- Lua 5.4 behavior:
--   create+resume error: TBC vars NOT closed (only on coroutine.close)
--   wrap error: TBC vars ARE closed before error propagates

-- create+resume: TBC vars not closed on error
do
  local closed = false
  local co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function() closed = true end})
    error("boom")
  end)
  local ok, err = coroutine.resume(co)
  assert(not ok)
  assert(not closed, "create+resume: TBC should NOT be closed on error")

  -- coroutine.close on dead coroutine should close them
  coroutine.close(co)
  assert(closed, "coroutine.close should close TBC vars")
end

-- create+resume with yield then error: TBC not closed
do
  local closed = false
  local co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function() closed = true end})
    coroutine.yield()
    error("boom")
  end)
  coroutine.resume(co)
  assert(not closed)
  coroutine.resume(co) -- triggers error
  assert(not closed, "create+resume+yield+error: TBC should NOT be closed")
end

-- wrap: TBC vars ARE closed on error
do
  local closed = false
  local co = coroutine.wrap(function()
    local x <close> = setmetatable({}, {__close = function() closed = true end})
    coroutine.yield()
    error("boom")
  end)
  co()
  assert(not closed)
  local ok, err = pcall(co)
  assert(not ok)
  assert(closed, "wrap: TBC should be closed on error propagation")
end

-- wrap with cascading __close errors
do
  local x = 0
  local co = coroutine.wrap(function()
    local xx <close> = setmetatable({}, {__close = function(_, msg)
      x = x + 1
      assert(string.find(msg, "@XXX"))
      error("@YYY")
    end})
    local xv <close> = setmetatable({}, {__close = function()
      x = x + 1
      error("@XXX")
    end})
    coroutine.yield(100)
    error(200)
  end)
  assert(co() == 100)
  assert(x == 0)
  local st, msg = pcall(co)
  assert(x == 2, "both __close handlers should have run, x=" .. x)
  assert(not st and string.find(msg, "@YYY"), "final error should be @YYY")
end

-- pcall inside coroutine still closes TBC vars
do
  local closed = false
  local co = coroutine.create(function()
    pcall(function()
      local x <close> = setmetatable({}, {__close = function() closed = true end})
      error("boom")
    end)
    assert(closed, "pcall inside coroutine should close TBC")
  end)
  coroutine.resume(co)
end

print("PASS")
