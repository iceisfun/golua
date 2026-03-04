-- Test: nextvar.lua - next on sliding window table
-- From: nextvar.lua
-- What: Tests next iteration on a table where elements are inserted at increasing keys and previous keys are deleted, leaving only one element.

do
  local a = {}
  for i=1,1000 do
    a[i] = i; a[i - 1] = undef
  end
  assert(next(a,nil) == 1000 and next(a,1000) == nil)

  assert(next({}) == nil)
  assert(next({}, nil) == nil)
end
