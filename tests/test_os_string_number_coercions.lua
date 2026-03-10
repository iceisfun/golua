do
  local ok, msg, errno = os.remove(123)
  assert(ok == nil)
  assert(type(msg) == "string")
  assert(string.find(msg, "123", 1, true), tostring(msg))
  assert(type(errno) == "number")
end

do
  local ok, msg, errno = os.rename(123, 456)
  assert(ok == nil)
  assert(type(msg) == "string")
  assert(string.find(msg, "123", 1, true), tostring(msg))
  assert(type(errno) == "number")
end
