-- Test: nextvar.lua - next with all kinds of keys
-- From: nextvar.lua (modified: removed io.stdin userdata key since io module not available)
-- What: Tests that next/pairs correctly iterates a table with integer, float, short string, long string, C function, Lua function, thread, boolean, and table keys.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local a = {
    [1] = 1,                        -- integer
    [1.1] = 2,                      -- float
    ['x'] = 3,                      -- short string
    [string.rep('x', 1000)] = 4,    -- long string
    [print] = 5,                    -- C function
    [checkerror] = 6,               -- Lua function
    [coroutine.running()] = 7,      -- thread
    [true] = 8,                     -- boolean
    [{}] = 9,                       -- table
  }
  local b = {}; for i = 1, 9 do b[i] = true end
  for k, v in pairs(a) do
    assert(b[v]); b[v] = undef
  end
  assert(next(b) == nil)        -- 'b' now is empty
end
