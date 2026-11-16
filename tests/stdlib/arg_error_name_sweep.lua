-- Test: argument type errors report the __name metafield (luaL_typeerror),
-- including for an absent argument when the nil type metatable has __name.
local up = 1
local function lf() return up end
local t = setmetatable({}, {__name = 'MyType'})

local cases = {
  {select, t},
  {os.date, t}, {os.remove, t}, {os.getenv, t}, {os.execute, t},
  {os.rename, t, 'a'}, {os.rename, 'a', t},
  {os.setlocale, t}, {os.setlocale, 'C', t},
  {require, t}, {load, t}, {load, 'x', t}, {load, 'x', 'n', t},
  {loadfile, t}, {loadfile, 'x', t}, {dofile, t},
  {coroutine.resume, t}, {coroutine.status, t}, {coroutine.close, t},
  {coroutine.isyieldable, t}, {coroutine.wrap, t}, {coroutine.create, t},
  {debug.getupvalue, t, 1}, {debug.setupvalue, t, 1, 1}, {debug.upvalueid, t, 1},
  {debug.upvaluejoin, t, 1, lf, 1}, {debug.upvaluejoin, lf, 1, t, 1},
  {debug.sethook, t, 'c'},
  {xpcall, print, t}, {tonumber, t, 10}, {warn, t}, {warn, 'a', t},
}
for i, c in ipairs(cases) do
  local ok, err = pcall(c[1], table.unpack(c, 2))
  assert(not ok, i)
  assert(err:find("got MyType)", 1, true), i .. ": " .. tostring(err))
end

-- an absent argument reads as nil, so the nil type metatable's __name applies
debug.setmetatable(nil, {__name = 'NilT'})
local function msg(f, ...)
  local ok, err = pcall(f, ...)
  assert(not ok)
  return err
end
local got = {
  msg(string.sub), msg(select), msg(math.floor),
  msg(coroutine.status), msg(xpcall, print), msg(load), msg(debug.getupvalue),
}
debug.setmetatable(nil, nil)
for i, err in ipairs(got) do
  assert(err:find("got NilT)", 1, true), i .. ": " .. err)
end

-- without a nil __name it is still "no value"
assert(msg(string.sub):find("got no value)", 1, true))
print("PASS")
