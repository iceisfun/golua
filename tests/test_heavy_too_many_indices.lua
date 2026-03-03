-- Test: heavy.lua - Too many table indices (memory exhaustion)
-- From: heavy.lua
-- What: Tests that inserting indices until memory exhaustion produces a catchable error

do
  print("creating too many table indices")
  local a = {}
  local st, msg = pcall(function ()
    for i = 1, math.huge do
      if i % (0x100000) == 0 then
        io.stderr:write("(", i // 2^20, " M)")
      end
      a[i] = i
     end
  end)
  print("\nmemory: ", collectgarbage'count' * 1024)
  print("expected error: ", msg)
  print("size:", #a)
end
