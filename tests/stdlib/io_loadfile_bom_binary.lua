-- Test: loadfile/dofile accept a binary chunk after a BOM and/or '#' line
-- What: reference luaL_loadfilex skips the BOM and the first '#' line and only
-- then looks for the binary signature

local name = os.tmpname()
local dump = string.dump(function() return "ok" end)

local function write(...)
  local f = assert(io.open(name, "wb"))
  f:write(...)
  f:close()
end

write("\xEF\xBB\xBF", dump)
assert(assert(loadfile(name))() == "ok")
assert(dofile(name) == "ok")

write("#!/usr/bin/lua\n", dump)
assert(assert(loadfile(name))() == "ok")

write("\xEF\xBB\xBF#!/usr/bin/lua\n", dump)
assert(assert(loadfile(name))() == "ok")
assert(dofile(name) == "ok")

-- mode "t" must reject the binary chunk
local fn, err = loadfile(name, "t")
assert(fn == nil and err:find("attempt to load a binary chunk", 1, true), err)

-- text after a BOM still works
write("\xEF\xBB\xBFreturn 7")
assert(assert(loadfile(name))() == 7)

os.remove(name)
