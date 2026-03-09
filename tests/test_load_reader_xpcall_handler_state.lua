-- xpcall handler state should not leak from load(reader) internals.

do
  local seen = {}
  local ok, kind = xpcall(function()
    local f, e = load(function()
      error({ code = 2 })
    end)
    -- Direct load(...) call stringifies non-string reader errors.
    assert(f == nil)
    assert(type(e) == "string")
    error(e)
  end, function(e)
    seen[#seen + 1] = type(e)
    return type(e)
  end)

  assert(not ok)
  -- Lua 5.4 calls the handler once for the internal reader error object,
  -- then again for the explicit error(e) with the string message.
  assert(#seen == 2, "expected 2 handler calls, got " .. tostring(#seen))
  assert(seen[1] == "table", "first handler arg type mismatch: " .. tostring(seen[1]))
  assert(seen[2] == "string", "second handler arg type mismatch: " .. tostring(seen[2]))
  assert(kind == "string", "xpcall should return second handler result")
end

-- xpcall body succeeds after reader error: handler called but success returned.
do
  local handler_calls = 0
  local ok, result = xpcall(function()
    local f, e = load(function() error("oops") end)
    return "success"
  end, function(e)
    handler_calls = handler_calls + 1
    return e
  end)

  assert(ok == true, "xpcall should succeed: " .. tostring(ok))
  assert(result == "success", "expected 'success', got: " .. tostring(result))
  -- Lua 5.4 calls the handler for the reader error even though xpcall succeeds.
  assert(handler_calls == 1, "expected 1 handler call, got " .. tostring(handler_calls))
end

-- pcall(load, reader) with non-string error preserves raw error object.
do
  local ok, fn, err = pcall(load, function() error({code=3}) end)
  assert(ok == true, "pcall(load,...) should succeed")
  assert(fn == nil, "load should return nil on error")
  assert(type(err) == "table", "error should be raw table, got " .. type(err))
  assert(err.code == 3, "error.code should be 3")
end

-- Multiple load(reader) errors under xpcall don't accumulate handler state.
do
  local seen = {}
  local ok, kind = xpcall(function()
    -- First failing load
    local f1, e1 = load(function() error({tag="first"}) end)
    assert(f1 == nil)
    -- Second failing load
    local f2, e2 = load(function() error({tag="second"}) end)
    assert(f2 == nil)
    -- Now raise the real error
    error("final")
  end, function(e)
    seen[#seen + 1] = type(e)
    return type(e)
  end)

  assert(not ok)
  -- Handler called once per reader error + once for final error.
  assert(#seen == 3, "expected 3 handler calls, got " .. tostring(#seen))
  assert(seen[1] == "table", "first: " .. tostring(seen[1]))
  assert(seen[2] == "table", "second: " .. tostring(seen[2]))
  assert(seen[3] == "string", "third: " .. tostring(seen[3]))
  assert(kind == "string", "should return last handler result")
end

-- pcall inside xpcall with load(reader) isolates handler state.
do
  local handler_calls = 0
  local ok, result = xpcall(function()
    local pok, fn, err = pcall(load, function() error("inner") end)
    assert(pok == true)
    assert(fn == nil)
    error("outer")
  end, function(e)
    handler_calls = handler_calls + 1
    return "handled:" .. tostring(e)
  end)

  assert(not ok)
  -- pcall clears MsgHandler, so the reader error inside pcall(load,...)
  -- should NOT trigger the xpcall handler. Only error("outer") triggers it.
  assert(handler_calls == 1, "expected 1 handler call, got " .. tostring(handler_calls))
  assert(type(result) == "string")
end
