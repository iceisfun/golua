-- Test: attrib.lua - require with invalid package.path type
-- From: attrib.lua
-- What: Ensures require errors properly when package.path is not a string

do
  local oldpath = package.path
  package.path = {}
  local s, err = pcall(require, "no-such-file")
  assert(not s and string.find(err, "package.path"))
  package.path = oldpath
end
