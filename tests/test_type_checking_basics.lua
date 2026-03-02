-- Test: calls.lua - Type checking basics
-- From: calls.lua
-- What: Tests the type() function with various value types

do
  assert(type(1<2) == 'boolean')
  assert(type(true) == 'boolean' and type(false) == 'boolean')
  assert(type(nil) == 'nil'
     and type(-3) == 'number'
     and type'x' == 'string'
     and type{} == 'table'
     and type(type) == 'function')
  assert(type(assert) == type(print))
  assert(not pcall(type))
end
