-- Test: heavy.lua - Control structure too long
-- From: heavy.lua
-- What: Tests that the compiler rejects a while loop exceeding the 24-bit jump offset limit.
--
-- The limit formula is ((1 << 24) - 2) // N where N is the number of
-- instructions emitted per "a = a + 1" statement.  Reference Lua 5.4.0-5.4.3
-- used N=3 (GETTABUP, ADDI, SETTABUP); Lua 5.4.4+ and GoLua use N=4
-- (GETTABUP, ADDI, MMBINI, SETTABUP).

do
  print("control structure too long")
  local lim = ((1 << 24) - 2) // 4
  local s = string.rep("a = a + 1\n", lim)
  s = "while true do " .. s .. "end"
  assert(load(s))
  print("ok with " .. lim .. " lines")
  lim = lim + 3
  s = string.rep("a = a + 1\n", lim)
  s = "while true do " .. s .. "end"
  local st, msg = load(s)
  assert(not st and string.find(msg, "too long"))
  print(msg)
end
