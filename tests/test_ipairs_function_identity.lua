-- Test: nextvar.lua - ipairs function identity
-- From: nextvar.lua
-- What: Tests that the ipairs iterator function is always the same object.

do
  assert(type(ipairs{}) == 'function' and ipairs{} == ipairs{})
end
