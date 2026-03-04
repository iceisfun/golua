-- Test: events.lua - Metatables for basic types
-- From: events.lua
-- What: Tests setting metatables on numbers, booleans, and nil via debug.setmetatable

do
  local debug = require 'debug'

  local mt = {__index = function (a,b) return a+b end,
        __len = function (x) return math.floor(x) end}
  debug.setmetatable(10, mt)
  assert(getmetatable(-2) == mt)
  assert((10)[3] == 13)
  assert((10)["3"] == 13)
  assert(#3.45 == 3)
  debug.setmetatable(23, nil)
  assert(getmetatable(-2) == nil)

  debug.setmetatable(true, mt)
  assert(getmetatable(false) == mt)
  mt.__index = function (a,b) return a or b end
  assert((true)[false] == true)
  assert((false)[false] == false)
  debug.setmetatable(false, nil)
  assert(getmetatable(true) == nil)

  debug.setmetatable(nil, mt)
  assert(getmetatable(nil) == mt)
  mt.__add = function (a,b) return (a or 1) + (b or 2) end
  assert(10 + nil == 12)
  assert(nil + 23 == 24)
  assert(nil + nil == 3)
  debug.setmetatable(nil, nil)
  assert(getmetatable(nil) == nil)
end
