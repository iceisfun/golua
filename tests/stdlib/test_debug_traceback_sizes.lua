-- Test: db.lua - Traceback sizes
-- From: db.lua
-- What: Tests that debug.traceback properly truncates long traces with "..." separator

do
  local function countlines(s)
    local n = 0
    for _ in string.gmatch(s, "\n") do n = n + 1 end
    return n + 1  -- count last line too
  end

  local function deep (lvl, n)
    if lvl == 0 then return (debug.traceback("message", n))
    else return (deep(lvl-1, n)) end
  end
  local function checkdeep (total, start)
    local s = deep(total, start)
    local rest = string.match(s, "^message\nstack traceback:\n(.*)$")
    local cl = countlines(rest)
    assert(cl <= 10 + 11 + 1)
  end
  for d = 1, 51, 10 do
    for l = 1, d do
      coroutine.wrap(checkdeep)(d, l)
    end
  end
end
