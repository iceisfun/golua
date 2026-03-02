-- Test: nextvar.lua - Length operator with nils
-- From: nextvar.lua
-- What: Tests the # operator on tables containing nil values at various positions.

do
  assert(#{} == 0)
  assert(#{nil} == 0)
  assert(#{nil, nil} == 0)
  assert(#{nil, nil, nil} == 0)
  assert(#{nil, nil, nil, nil} == 0)
  assert(#{1, 2, 3, nil, nil} == 3)
end
