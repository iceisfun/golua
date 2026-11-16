-- Test: the ipairs iterator checks its control value
-- What: reference ipairsaux uses luaL_checkinteger on argument 2

local f = ipairs({})
local function bad(...)
  local ok, err = pcall(f, ...)
  assert(not ok, "expected an error")
  return err
end

assert(bad({1, 2}, nil):find("bad argument #2", 1, true))
assert(bad({1, 2}, nil):find("number expected, got nil", 1, true))
assert(bad({1, 2}):find("number expected, got no value", 1, true))
assert(bad({1, 2}, 1.5):find("number has no integer representation", 1, true))
assert(bad({1, 2}, {}):find("number expected, got table", 1, true))

-- valid control values still work (including numeric strings and floats)
local i, v = f({10, 20}, 0)
assert(i == 1 and v == 10)
i, v = f({10, 20}, 1.0)
assert(i == 2 and v == 20)
i, v = f({10, 20}, "1")
assert(i == 2 and v == 20)
assert(f({10, 20}, 2) == nil)
