-- Test: with a __name on the nil type metatable, an absent or nil argument
-- is reported by that name (luaL_typeerror checks __name before falling back
-- to "no value" or "nil"), including in functions that check the absent case
-- themselves.
local tmpname = os.tmpname()
local fh = assert(io.open(tmpname, 'w'))
debug.setmetatable(nil, {__name = 'NilT'})
local function msg(f, ...)
  local ok, err = pcall(f, ...)
  assert(not ok)
  return err
end
local got = {
  msg(os.getenv), msg(os.getenv, nil),
  msg(os.remove), msg(os.remove, nil),
  msg(os.rename), msg(os.rename, nil), msg(os.rename, 'a'), msg(os.rename, 'a', nil),
  msg(io.open), msg(io.open, nil),
  msg(warn), msg(require), msg(require, nil),
  msg(setmetatable, {}),
  msg(table.create),
  msg(fh.setvbuf, fh), msg(fh.setvbuf, fh, nil),
  msg(fh.close, nil),
}
debug.setmetatable(nil, nil)
for i, err in ipairs(got) do
  assert(err:find("got NilT)", 1, true), i .. ": " .. err)
end

-- without a nil __name the absent and nil cases are still told apart
assert(msg(os.getenv):find("got no value)", 1, true))
assert(msg(os.getenv, nil):find("got nil)", 1, true))
assert(msg(io.open):find("got no value)", 1, true))
assert(msg(io.open, nil):find("got nil)", 1, true))
assert(msg(warn):find("got no value)", 1, true))
assert(msg(fh.setvbuf, fh):find("got no value)", 1, true))
fh:close()
os.remove(tmpname)
print("PASS")
