-- Test: attrib.lua - package.searchpath with large paths
-- From: attrib.lua
-- What: Tests package.searchpath with paths containing many templates and very long templates

do
  local max = _soft and 100 or 2000
  local t = {}
  for i = 1,max do t[i] = string.rep("?", i%10 + 1) end
  t[#t + 1] = ";"
  local path = table.concat(t, ";")
  local s, err = package.searchpath("xuxu", path)
  assert(not s and
         string.find(err, string.rep("xuxu", 10)) and
         #string.gsub(err, "[^\n]", "") >= max)
  local path = string.rep("?", max)
  local s, err = package.searchpath("xuxu", path)
  assert(not s and string.find(err, string.rep('xuxu', max)))
end
