-- Test: cstack.lua - Nesting coroutines after recoverable errors
-- From: cstack.lua
-- What: Tests stack overflow when nesting coroutines inside pcall error recovery (bug in 5.4.2)

do
  local function checkerror(expected, f, ...)
    local ok, msg = pcall(f, ...)
    assert(not ok and type(msg) == "string" and string.find(msg, expected),
           "expected error '" .. expected .. "' got: " .. tostring(msg))
  end

  local count = 0
  local function foo()
    count = count + 1
    pcall(1)   -- create an error
    coroutine.wrap(foo)()
  end
  checkerror("stack overflow", foo)
end
