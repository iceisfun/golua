-- Test: nextvar.lua - Clear global table
-- From: nextvar.lua
-- What: Tests clearing non-essential entries from the global table, keeping loaded packages, functions, and uppercase/underscore names.

do   -- clear global table
  local a = {}
  for n,v in pairs(_G) do a[n]=v end
  for n,v in pairs(a) do
    if not package.loaded[n] and type(v) ~= "function" and
       not string.find(n, "^[%u_]") then
      _G[n] = undef
    end
    collectgarbage()
  end
end
