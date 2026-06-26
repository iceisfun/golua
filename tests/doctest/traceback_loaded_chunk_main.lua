-- A loaded chunk's main function (linedefined == 0) must be labeled
-- "in main chunk" in a traceback even when it is NOT the outermost frame —
-- e.g. reached through pcall/xpcall, a C call boundary, with no name available.
-- golua previously gated the "main chunk" label on the outermost frame only, so
-- such a frame was mislabeled "in function '?'". The label is a *fallback*: a
-- main chunk reached via a named local still shows that name.

-- pcall/xpcall'd loaded chunk at a non-outermost frame -> "in main chunk":
do
  local f = load("error('boom')", "=[chunk]")
  local _, tb = xpcall(f, debug.traceback)
  print(tb:find("%[chunk%]:1: in main chunk") ~= nil)
  --> =true
  print(tb:find("in function '%?'") ~= nil)
  --> =false
end

-- A loaded chunk reached via a name still shows that name (here 'f' is an
-- upvalue of the wrapping function), NOT "main chunk":
do
  local f = load("error('boom')", "=[chunk]")
  local _, tb = xpcall(function() f() end, debug.traceback)
  print(tb:find("%[chunk%]:1: in upvalue 'f'") ~= nil)
  --> =true
end
