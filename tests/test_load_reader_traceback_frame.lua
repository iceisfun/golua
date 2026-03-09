-- Direct load(reader) errors should include the reader callback frame
-- in traceback text (Lua 5.4 behavior).

do
  local fn, err = load(function()
    error("oops")
  end)

  assert(fn == nil)
  assert(type(err) == "string")
  assert(string.find(err, "stack traceback", 1, true) ~= nil)
  assert(string.find(err, "in function <", 1, true) ~= nil,
    "expected reader callback frame in traceback: " .. tostring(err))
end

do
  local fn, err = load(function()
    error({ tag = "T" })
  end)

  assert(fn == nil)
  assert(type(err) == "string")
  assert(string.find(err, "stack traceback", 1, true) ~= nil)
  assert(string.find(err, "in function <", 1, true) ~= nil,
    "expected reader callback frame in traceback for non-string error")
end
