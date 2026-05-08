-- Test: Lua 5.5 coroutine.close(coroutine.running()) self-terminates the
-- running coroutine via long-jump rather than raising "cannot close a
-- running coroutine".  Pending <close> handlers run on the way out, and
-- the resumer sees (true, nil).

-- Bare self-close: silently terminates, resume returns ok
do
  local trace = {}
  local co = coroutine.create(function()
    table.insert(trace, "before")
    coroutine.close(coroutine.running())
    table.insert(trace, "UNREACHABLE")
  end)
  local ok, err = coroutine.resume(co)
  assert(ok == true, "resume ok=" .. tostring(ok))
  assert(err == nil, "resume err=" .. tostring(err))
  assert(coroutine.status(co) == "dead", "status=" .. coroutine.status(co))
  assert(trace[1] == "before" and trace[2] == nil,
    "unexpected trace: " .. table.concat(trace, ","))
end

-- Pending <close> handlers run during self-close
do
  local trace = {}
  local co = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function()
      table.insert(trace, "close")
    end})
    table.insert(trace, "before")
    coroutine.close(coroutine.running())
  end)
  local ok = coroutine.resume(co)
  assert(ok == true)
  assert(trace[1] == "before" and trace[2] == "close",
    "expected before/close, got: " .. table.concat(trace, ","))
end

-- pcall does NOT catch the self-close: long-jump skips the protected call
do
  local co = coroutine.create(function()
    pcall(coroutine.close, coroutine.running())
    error("UNREACHABLE")
  end)
  local ok = coroutine.resume(co)
  assert(ok == true, "pcall around self-close should not be catchable")
  assert(coroutine.status(co) == "dead")
end

-- coroutine.close() with no argument defaults to current coroutine on 5.5
do
  local co = coroutine.create(function()
    coroutine.close()
    error("UNREACHABLE")
  end)
  local ok = coroutine.resume(co)
  assert(ok == true)
end

print("OK")
