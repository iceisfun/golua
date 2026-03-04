-- Test: heavy.lua - Too many strings (memory exhaustion)
-- From: heavy.lua
-- What: Tests that creating strings in a tight loop until memory exhaustion produces a catchable error

do
  print("creating too many strings")
  local a = {}
  local st, msg = pcall(function ()
    for i = 1, math.huge do
      if i % (0x100000) == 0 then
        io.stderr:write("(", i // 2^20, " M)")
      end
      a[i] = string.pack("I", i)
    end
  end)
  local size = #a
  a = collectgarbage'count'
  print("\nmemory:", a * 1024)
  print("expected error:", msg)
  print("size:", size)
end
