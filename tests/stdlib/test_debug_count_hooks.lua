-- Test: db.lua - Count hooks
-- From: db.lua
-- What: Tests debug.sethook with count mode (fires every N instructions)

do
  local a=0
  debug.sethook(function (e) a=a+1 end, "", 1)
  a=0; for i=1,1000 do end; assert(1000 < a and a < 1012)
  debug.sethook(function (e) a=a+1 end, "", 4)
  a=0; for i=1,1000 do end; assert(250 < a and a < 255)
  debug.sethook()
end
