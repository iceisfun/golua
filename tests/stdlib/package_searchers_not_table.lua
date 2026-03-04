-- Test: attrib.lua - package.searchers not a table
-- From: attrib.lua
-- What: Ensures require errors when package.searchers is not a table

do
  local searchers = package.searchers
  package.searchers = 3
  local st, msg = pcall(require, 'a')
  assert(not st and string.find(msg, "must be a table"))
  package.searchers = searchers
end
