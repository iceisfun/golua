-- Test: db.lua - debug.traceback
-- From: db.lua
-- What: Tests debug.traceback with various arguments and truncation behavior

do
  assert(debug.traceback(print) == print)
  assert(debug.traceback(print, 4) == print)
  assert(string.find(debug.traceback("hi", 4), "^hi\n"))
  assert(string.find(debug.traceback("hi"), "^hi\n"))
  assert(string.find(debug.traceback(), "^stack traceback:\n"))
end
