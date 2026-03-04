-- Test: errors.lua - Basic error messages
-- From: errors.lua
-- What: Tests error() with level 0 (no source info), error() with no args, and common runtime errors

do
  local function doit(s)
    local f, msg = load(s)
    if not f then return msg end
    local ok, err = pcall(f)
    if not ok then return err end
    return nil
  end

  assert(doit("error('hi', 0)") == 'hi')
  assert(doit("error()") == nil)
  assert(doit("table.unpack({}, 1, n=2^30)"))
  assert(doit("a=math.sin()"))
  assert(not doit("tostring(1)") and doit("tostring()"))
  assert(doit"tonumber()")
  assert(doit"assert(false)")
  assert(doit"assert(nil)")
end
