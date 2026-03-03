-- Test: errors.lua - Stack overflow errors
-- From: errors.lua
-- What: Tests stack overflow detection, recovery, and error reporting

do
  local function doit(s)
    local f, msg = load(s)
    if not f then return msg end
    local ok, err = pcall(f)
    if not ok then return err end
    return nil
  end

  local function checkstackmessage(msg)
    return (string.find(msg, "stack overflow") ~= nil)
  end

  collectgarbage()
  local C = 0
  local function auxy () C=C+1; auxy() end
  function YY ()
    collectgarbage("stop")
    auxy()
    collectgarbage("restart")
  end
  assert(checkstackmessage(doit('YY()')))
  assert(checkstackmessage(doit('YY()')))
end
