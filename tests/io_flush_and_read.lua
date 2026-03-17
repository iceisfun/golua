do
  assert(type(io.flush) == "function", "io.flush should be available when io is enabled")
  assert(io.flush() == true, "io.flush should return true on success")
end

do
  local f = assert(io.tmpfile())
  assert(f:flush() == true, "file:flush should return true on success")
end

do
  local f = assert(io.tmpfile())
  local ok, err = pcall(function()
    return f:read({})
  end)
  assert(not ok, "file:read should reject non-string/non-number formats")
  assert(type(err) == "string" and string.find(err, "bad argument #2 to '?'", 1, true), tostring(err))
  assert(string.find(err, "string expected, got table", 1, true), tostring(err))
end
