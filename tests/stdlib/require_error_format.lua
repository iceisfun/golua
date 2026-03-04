-- Test: attrib.lua - require error message format
-- From: attrib.lua
-- What: Checks the exact format of 'module not found' error messages from require

do
  local oldpath = package.path
  local oldcpath = package.cpath
  package.path = "?.lua;?/?"
  package.cpath = "?.so;?/init"
  local st, msg = pcall(require, 'XXX')
  local expected = [[module 'XXX' not found:
	no field package.preload['XXX']
	no file 'XXX.lua'
	no file 'XXX/XXX'
	no file 'XXX.so'
	no file 'XXX/init']]
  assert(msg == expected)
  package.path = oldpath
  package.cpath = oldcpath
end
