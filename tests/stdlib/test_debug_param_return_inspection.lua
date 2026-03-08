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

  -- Multi-return: string.byte("ABCDE",1,-1) returns 65,66,67,68,69
  -- Return hook should expose args first, then return values.
  debug.sethook(hook, "cr")
  on = true; string.byte("ABCDE", 1, -1); on = false
  debug.sethook()
  eqseq(inp, {"ABCDE", 1, -1})
  eqseq(out, {65, 66, 67, 68, 69})

  -- Also verify full getlocal view during return hook: args then results
  local retLocals, retFt, retNt
  local function hook2(event)
    if not on then return end
    local ar = debug.getinfo(2, "ruS")
    if ar.what ~= "C" then return end
    if event == "return" then
      retFt = ar.ftransfer
      retNt = ar.ntransfer
      retLocals = {}
      for i = 1, 20 do
        local n, v = debug.getlocal(2, i)
        if not n then break end
        retLocals[i] = v
      end
    end
  end
  debug.sethook(hook2, "r")
  on = true; string.byte("ABCDE", 1, -1); on = false
  debug.sethook()
  -- ftransfer should point past the 3 args, ntransfer = 5 return values
  assert(retFt == 4, "ftransfer expected 4, got " .. tostring(retFt))
  assert(retNt == 5, "ntransfer expected 5, got " .. tostring(retNt))
  -- Args at indices 1..3, return values at indices 4..8
  eqseq({retLocals[1], retLocals[2], retLocals[3]}, {"ABCDE", 1, -1})
  eqseq({retLocals[4], retLocals[5], retLocals[6], retLocals[7], retLocals[8]},
        {65, 66, 67, 68, 69})

  -- Call-hook argument mutation must be visible in return-hook locals.
  -- Mutate args during call hook, verify return hook sees mutated args.
  local mutRetLocals
  local function hook3(event)
    if not on then return end
    local ar = debug.getinfo(2, "rnuS")
    if ar.what ~= "C" or ar.name ~= "byte" then return end
    if event == "call" then
      debug.setlocal(2, 1, "XYZ")
      debug.setlocal(2, 2, 1)
      debug.setlocal(2, 3, -1)
    elseif event == "return" then
      mutRetLocals = {}
      for i = 1, 20 do
        local n, v = debug.getlocal(2, i)
        if not n then break end
        mutRetLocals[i] = v
      end
    end
  end
  debug.sethook(hook3, "cr")
  on = true; string.byte("ABCDE", 2, 3); on = false
  debug.sethook()
  -- Return-hook args should reflect call-hook mutations
  eqseq({mutRetLocals[1], mutRetLocals[2], mutRetLocals[3]}, {"XYZ", 1, -1})
  -- Return values come from mutated args: string.byte("XYZ",1,-1) = 88,89,90
  eqseq({mutRetLocals[4], mutRetLocals[5], mutRetLocals[6]}, {88, 89, 90})
end
