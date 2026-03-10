assert(type(io.popen) == "function")

do
  local f = assert(io.popen("printf 'hello\\nworld\\n'", "r"))
  assert(io.type(f) == "file")
  assert(f:read("l") == "hello")
  assert(f:read("l") == "world")
  local ok, how, code = f:close()
  assert(ok == true)
  assert(how == "exit")
  assert(code == 0)
end

do
  local f = assert(io.popen("cat >/dev/null", "w"))
  assert(f:write("payload\n") == f)
  local ok, how, code = f:close()
  assert(ok == true)
  assert(how == "exit")
  assert(code == 0)
end

do
  local f = assert(io.popen("exit 7", "r"))
  local ok, how, code = f:close()
  assert(ok == nil)
  assert(how == "exit")
  assert(code == 7)
end
