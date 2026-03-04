-- Test: files.lua - loadfile tests
-- From: files.lua
-- What: Tests loadfile with various scenarios: empty file, comments, BOM, environments

do
  local file = os.tmpname()

  local function testloadfile(content, expected)
    local f = assert(io.open(file, "wb"))
    if content then f:write(content) end
    f:close()
    local fn = assert(loadfile(file))
    local result = fn()
    assert(result == expected,
      string.format("expected %s, got %s", tostring(expected), tostring(result)))
  end

  -- loading empty file
  testloadfile(nil, nil)
  -- loading file with initial comment
  testloadfile("# a non-ending comment", nil)
  -- checking Unicode BOM
  testloadfile("\xEF\xBB\xBF# some comment\nreturn 234", 234)
  testloadfile("\xEF\xBB\xBFreturn 239", 239)
  -- loading file with return value
  testloadfile("return 42", 42)

  -- loadfile with environment
  local f = assert(io.open(file, "w"))
  f:write("return X")
  f:close()
  local fn = assert(loadfile(file, "t", {X = 999}))
  assert(fn() == 999)

  -- loadfile mode restriction
  f = assert(io.open(file, "w"))
  f:write("return 1")
  f:close()
  -- text mode should work for text files
  local fn2 = assert(loadfile(file, "t"))
  assert(fn2() == 1)

  os.remove(file)
end
