-- To-be-closed variables must still be closed when a call expression in
-- return position fails during call dispatch (before OP_RETURN runs).

local function contains(s, needle)
  return type(s) == "string" and string.find(s, needle, 1, true) ~= nil
end

do
  local log = {}
  local function mk(name, fail)
    return setmetatable({}, {
      __close = function(_, err)
        log[#log + 1] = name .. ":" .. tostring(err)
        if fail then error("E-" .. name) end
      end,
    })
  end

  local function f()
    local a <close> = mk("a", false)
    local b <close> = mk("b", false)
    return nosuch()
  end

  local ok, err = pcall(f)
  assert(ok == false)
  assert(contains(tostring(err), "attempt to call a nil value"))
  assert(#log == 2, "expected both close handlers to run")
  assert(contains(log[1], "attempt to call a nil value"))
  assert(contains(log[2], "attempt to call a nil value"))
end

do
  local log = {}
  local function mk(name, fail)
    return setmetatable({}, {
      __close = function(_, err)
        log[#log + 1] = name .. ":" .. tostring(err)
        if fail then error("E-" .. name) end
      end,
    })
  end

  local function f()
    local a <close> = mk("a", false)
    local b <close> = mk("b", true)
    return nosuch()
  end

  local ok, err = pcall(f)
  assert(ok == false)
  assert(contains(tostring(err), "E-b"), "close error should replace call error")
  assert(#log == 2, "expected close chain to continue under pcall")
  assert(contains(log[1], "attempt to call a nil value"))
  assert(contains(log[2], "E-b"))
end
