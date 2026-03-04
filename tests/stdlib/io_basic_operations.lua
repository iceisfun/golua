-- Test: files.lua - Basic I/O operations
-- From: files.lua
-- What: Tests io.input, io.output, io.stdin/stdout/stderr, io.type, io.open errors

do
  assert(type(os.getenv"PATH") == "string")

  assert(io.input(io.stdin) == io.stdin)
  assert(not pcall(io.input, "non-existent-file"))
  assert(io.output(io.stdout) == io.stdout)

  -- cannot close standard files
  assert(not io.close(io.stdin) and
         not io.stdout:close() and
         not io.stderr:close())

  assert(type(io.input()) == "userdata" and io.type(io.output()) == "file")
  assert(type(io.stdin) == "userdata" and io.type(io.stderr) == "file")
  assert(not io.type(8))
  local a = {}; setmetatable(a, {})
  assert(not io.type(a))

  -- io.open error returns
  local a, b, c = io.open('xuxu_nao_existe')
  assert(not a and type(b) == "string" and type(c) == "number")

  a, b, c = io.open('/a/b/c/d', 'w')
  assert(not a and type(b) == "string" and type(c) == "number")
end
