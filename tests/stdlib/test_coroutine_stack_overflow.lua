-- Test: coroutine.lua - Stack overflow in coroutines
-- From: coroutine.lua
-- What: Tests that stack overflow is properly detected in coroutines with large table.unpack

do
  if not _soft then
    local lim = 1000000
    local t = {lim - 10, lim - 5, lim - 1, lim, lim + 1, lim + 5}
    for i = 1, #t do
      local j = t[i]
      local co = coroutine.create(function()
        return table.unpack({}, 1, j)
      end)
      local r, msg = coroutine.resume(co)
      assert(j < lim or not r)
    end
  end
end
