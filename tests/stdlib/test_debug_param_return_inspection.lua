-- Test: db.lua - Inspection of parameters/returned values
-- From: db.lua
-- What: Tests ftransfer/ntransfer fields in debug.getinfo for inspecting call/return values

do
  local function eqseq(a, b)
    assert(#a == #b)
    for i = 1, #a do assert(a[i] == b[i]) end
  end

  local on = false
  local inp, out
  local function hook (event)
    if not on then return end
    local ar = debug.getinfo(2, "ruS")
    local t = {}
    for i = ar.ftransfer, ar.ftransfer + ar.ntransfer - 1 do
      local _, v = debug.getlocal(2, i)
      t[#t + 1] = v
    end
    if event == "return" then out = t else inp = t end
  end
  debug.sethook(hook, "cr")
  on = true; math.sin(3); on = false
  debug.sethook()
  eqseq(inp, {3}); eqseq(out, {math.sin(3)})
end
