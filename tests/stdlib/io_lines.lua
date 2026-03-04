-- Test: files.lua - io.lines
-- From: files.lua
-- What: Tests line iteration with io.lines, including multiple formats

do
  local file = os.tmpname()

  -- Write test data
  local f = assert(io.open(file, "w"))
  f:write("line 1\n")
  f:write("line 2\n")
  f:write("line 3\n")
  f:write("line 4\n")
  f:write("line 5\n")
  f:write("line 6\n")
  f:close()

  -- Count lines using io.lines as iterator factory
  local n = 0
  local iter = io.lines(file)
  while iter() do n = n + 1 end
  assert(n == 6)

  -- Iterator should be closed after exhaustion
  local ok, err = pcall(iter)
  assert(not ok and string.find(err, "closed"))

  -- for loop with io.lines
  n = 0
  for l in io.lines(file) do
    n = n + 1
  end
  assert(n == 6)

  os.remove(file)
end
