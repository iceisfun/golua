-- Test: naming t[k] when the key k is in a register
-- What: reference rname names such an access after the key only when the
-- register provably holds a string constant. golua scanned back for any LOADK
-- into the key register, so a local key was named after its initializer, even
-- when it had since been reassigned, and an empty string constant, which it
-- used as its "no name" marker, came out as '?'.

local function errmsg(f, ...)
  local ok, msg = pcall(f, ...)
  assert(not ok)
  return msg
end

local function check(name, f, ...)
  local msg = errmsg(f, ...)
  assert(msg:find("(field '" .. name .. "')", 1, true), msg)
end

local t = {}

-- A string constant key is named, the empty string included.
check("", function() return t[""] + 1 end)
check("", function() local l = {} return l[""] + 1 end)
check("", function() t[""]() end)
check("y", function() return t["y"] + 1 end)
local KS <const> = "kk"
check("kk", function() return t[KS] + 1 end)

-- A key held in a local is a variable: no name.
check("?", function() local m = {}; local s2 = "k2"; return m[s2].z end)
check("?", function() local m = {}; local s2 = "k2"; m[s2]() end)
check("?", function(c) local m = {}; local s2 = "k2"; if c then s2 = "other" end; return m[s2].z end, true)
check("?", function() local k = "z" return t[k] + 1 end)
check("?", function() local k = "q" return t[k][2] end)
check("?", function() local k = "s" t[k]() end)

-- Neither is an upvalue or a key chosen at run time.
local up = "upk"
check("?", function() return t[up] + 1 end)
check("?", function() return t[1 < 2 and "a" or "b"] + 1 end)
check("?", function() return t[t.a or "dflt"] + 1 end)

-- With more than 256 constants a field read takes a register key too.
do
  local src = {"local t = {} "}
  for i = 1, 300 do src[#src + 1] = ("t.c%d = 1 "):format(i) end
  local function run(tail)
    src[#src + 1] = tail
    local msg = errmsg(assert(load(table.concat(src))))
    src[#src] = nil
    return msg
  end
  assert(run("return t.zz_missing + 1"):find("(field 'zz_missing')", 1, true))
  assert(run("return t[''] + 1"):find("(field '')", 1, true))
  assert(run("local k = 'kk' return t[k] + 1"):find("(field '?')", 1, true))
end

-- A traceback names a function called through a local key the same way.
do
  local m = {k2 = function() error("boom") end}
  local _, tb = xpcall(function() local s2 = "k2"; m[s2]() end, debug.traceback)
  assert(tb:find("in field '?'", 1, true) and not tb:find("in field 'k2'", 1, true), tb)
  _, tb = xpcall(function() m["k2"]() end, debug.traceback)
  assert(tb:find("in field 'k2'", 1, true), tb)
end
