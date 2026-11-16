-- Test: require accepts a precompiled module after a BOM and/or '#' line
-- What: the Lua searcher loads files like luaL_loadfilex, which skips the BOM
-- and the first '#' line before looking for the binary signature

local base = os.tmpname()
os.remove(base)
local dump = string.dump(function() return "modok" end)

local paths = {}
local function write(name, prefix)
  local path = base .. "_" .. name .. ".lua"
  local f = assert(io.open(path, "wb"))
  f:write(prefix, dump)
  f:close()
  paths[#paths + 1] = path
end

write("rsh", "#!x\n")
write("rbom", "\xEF\xBB\xBF")
write("rboth", "\xEF\xBB\xBF#!x\n")

local old = package.path
package.path = base .. "_?.lua"
for _, n in ipairs({"rsh", "rbom", "rboth"}) do
  local ok, r = pcall(require, n)
  assert(ok and r == "modok", tostring(r))
end
package.path = old
for _, p in ipairs(paths) do os.remove(p) end
