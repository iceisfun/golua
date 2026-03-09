-- load(reader) under a surrounding pcall wrapper should preserve raw reader
-- error values (matching Lua 5.4), unlike direct load(...) calls.

do
  local ok, fn, err = pcall(function()
    return load(function()
      error({ tag = "table" })
    end)
  end)

  assert(ok == true)
  assert(fn == nil)
  assert(type(err) == "table", "expected raw table error, got " .. type(err))
  assert(err.tag == "table")
end

do
  local ok, fn, err = pcall(function()
    return load(function()
      error("oops")
    end)
  end)

  assert(ok == true)
  assert(fn == nil)
  assert(type(err) == "string")
  assert(string.find(err, "oops", 1, true) ~= nil)
  assert(string.find(err, "stack traceback", 1, true) == nil,
    "wrapped pcall load should keep raw string error")
end
