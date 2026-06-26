-- Reference Lua never applies a file:line: prefix to a memory error
-- (LUA_ERRMEM is bare). golua's string-builder caps must match: a caught
-- "not enough memory" must have no location prefix.
do
  local ok, err = pcall(function() return ("x"):rep(1 << 40) end)
  print(ok, err)
  --> =false	not enough memory
end
