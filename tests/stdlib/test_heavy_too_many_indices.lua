-- Test: heavy.lua - Too many table indices (memory exhaustion)
-- From: heavy.lua
-- What: Tests that inserting indices until memory exhaustion produces a catchable error

do
  local a = {}
  local st, msg = pcall(function ()
    for i = 1, math.huge do
      if i % (0x100000) == 0 and io and io.stderr then
        io.stderr:write("(", i // 2^20, " M)")
      end
      a[i] = i
     end
  end)
end
