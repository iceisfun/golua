-- xpcall message-handler transformations must propagate through
-- to-be-closed error chaining.

local function has(s, needle)
  return type(s) == "string" and string.find(s, needle, 1, true) ~= nil
end

do
  local log = {}
  local function mk(name, fail)
    return setmetatable({}, {
      __close = function(_, err)
        log[#log + 1] = name .. ":" .. tostring(err)
        if fail then error("E" .. name) end
      end,
    })
  end

  local function f()
    local a <close> = mk("a", true)
    local b <close> = mk("b", true)
    return nosuch()
  end

  local ok, msg = xpcall(f, function(e)
    return "H:" .. tostring(e)
  end)

  assert(ok == false)
  assert(has(msg, "H:"), "xpcall result should be handler-transformed")
  assert(has(msg, "Ea"), "final error should come from last close error")

  assert(#log == 2, "expected two close handler calls")
  assert(has(log[1], "b:H:"), "first close arg should be handler-transformed")
  assert(has(log[2], "a:H:"), "second close arg should be handler-transformed")
  assert(has(log[2], "Eb"), "second close should receive transformed first close error")
end
