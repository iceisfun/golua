-- Test: attrib.lua - Package table types
-- From: attrib.lua
-- What: Checks that package.path, package.cpath, package.loaded, package.preload, and package.config have correct types

do
  assert(type(package.path) == "string")
  assert(type(package.cpath) == "string")
  assert(type(package.loaded) == "table")
  assert(type(package.preload) == "table")
  assert(type(package.config) == "string")
  assert(type(package.loadlib) == "nil")
end
