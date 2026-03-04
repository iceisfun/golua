-- Test: attrib.lua - package.preload
-- From: attrib.lua
-- What: Tests the package.preload mechanism for pre-registering module loaders

do
  local p = package
  package = {}
  p.preload.pl = function (...)
    local _ENV = {...}
    function xuxu (x) return x+20 end
    return _ENV
  end
  local pl, ext = require"pl"
  assert(require"pl" == pl)
  assert(pl.xuxu(10) == 30)
  assert(pl[1] == "pl" and pl[2] == ":preload:" and ext == ":preload:")
  package = p
  assert(type(package.path) == "string")
end
