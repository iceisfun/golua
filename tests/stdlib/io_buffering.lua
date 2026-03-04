-- Test: files.lua - File buffering (setvbuf)
-- From: files.lua
-- What: Tests full, no, and line buffering modes for file output

do
  local file = os.tmpname()

  local f = assert(io.open(file, "w"))
  local fr = assert(io.open(file, "r"))
  assert(f:setvbuf("full", 2000))
  f:write("x")
  assert(fr:read("all") == "")  -- full buffer; output not written yet
  f:close()
  fr:seek("set")
  assert(fr:read("all") == "x")   -- close flushes it
  fr:close()

  os.remove(file)
end
