-- Test: To-be-closed x compile-time constants
-- From: locals.lua
-- What: Tests interaction between to-be-closed variables and compile-time constant locals, verifying that the compiler does not confuse compile levels with stack levels.

do
  local flag = false
  local x = setmetatable({},
    {__close = function() assert(flag == false); flag = true end})
  local y <const> = nil
  local z <const> = nil
  do
      local a <close> = x
  end
  assert(flag)   -- 'x' must be closed here
end

do
  -- similar problem, but with implicit close in for loops
  local flag = false
  local x = setmetatable({},
    {__close = function () assert(flag == false); flag = true end})
  -- return an empty iterator, nil, nil, and 'x' to be closed
  local function a ()
    return (function () return nil end), nil, nil, x
  end
  local v <const> = 1
  local w <const> = 1
  local x <const> = 1
  local y <const> = 1
  local z <const> = 1
  for k in a() do
      a = k
  end    -- ending the loop must close 'x'
  assert(flag)   -- 'x' must be closed here
end
