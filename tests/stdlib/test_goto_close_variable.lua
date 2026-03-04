-- Test: goto.lua - Backward goto out of scope of close variable
-- From: goto.lua
-- What: Tests that jumping backward out of the scope of a <close> variable triggers __close

do
  local X
  goto L1

  ::L2:: goto L3

  ::L1:: do
    local a <close> = setmetatable({}, {__close = function () X = true end})
    assert(X == nil)
    if a then goto L2 end
  end

  ::L3:: assert(X == true)
end
