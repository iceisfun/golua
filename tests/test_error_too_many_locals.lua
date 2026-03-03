-- Test: errors.lua - Too many local variables error
-- From: errors.lua
-- What: Tests compiler limit on number of local variables (200)

do
  local s = "\nfunction foo ()\n  local "
  for i = 1, 300 do
    s = s .. "a" .. i .. ", "
  end
  s = s .. "a = 1\nend"
  local a, b = load(s)
  assert(string.find(b, "too many local variables"))
end
