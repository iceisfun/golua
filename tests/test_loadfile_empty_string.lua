do
  local f, err = loadfile("")
  assert(f == nil)
  assert(type(err) == "string")
  assert(string.find(err, "cannot open ", 1, true), tostring(err))
end

do
  local ok, err = pcall(dofile, "")
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "cannot open ", 1, true), tostring(err))
end
