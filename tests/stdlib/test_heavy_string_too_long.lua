-- Test: heavy.lua - String too long
-- From: heavy.lua
-- What: Tests that repeated string concatenation eventually produces a
--       "string length overflow" or "not enough memory" error

do
  print("creating a string too long")
  local a = "x"
  local st, msg = pcall(function ()
    while true do
      a = a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       .. a .. a.. a.. a.. a.. a.. a.. a.. a.. a
       print(string.format("string with %d bytes", #a))
    end
  end)
  assert(not st and
    (string.find(msg, "string length overflow") or
     string.find(msg, "not enough memory")))
  print("string length overflow with " .. #a * 100)
  print('+')
end
