-- Lua 5.4 default debug.getinfo() includes 'r' fields.

local info_func = debug.getinfo(function() end)
assert(info_func.ftransfer == 0, "expected ftransfer=0 for function form")
assert(info_func.ntransfer == 0, "expected ntransfer=0 for function form")

local function probe_level()
  local info = debug.getinfo(1)
  return info.ftransfer, info.ntransfer
end

local ftransfer, ntransfer = probe_level()
assert(ftransfer == 0, "expected ftransfer=0 for level form")
assert(ntransfer == 0, "expected ntransfer=0 for level form")

local co = coroutine.create(function()
  coroutine.yield()
end)
assert(coroutine.resume(co))

local info_thread = debug.getinfo(co, 1)
assert(info_thread.ftransfer == 0, "expected ftransfer=0 for thread form")
assert(info_thread.ntransfer == 0, "expected ntransfer=0 for thread form")

print("PASS")
