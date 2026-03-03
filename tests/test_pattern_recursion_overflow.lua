-- Test: cstack.lua - Recursion inside pattern matching
-- From: cstack.lua
-- What: Tests C-stack overflow detection in recursive pattern matching

do
  local function checkerror(expected, f, ...)
    local ok, msg = pcall(f, ...)
    assert(not ok and type(msg) == "string" and string.find(msg, expected),
           "expected error '" .. expected .. "' got: " .. tostring(msg))
  end

  local function f (size)
    local s = string.rep("a", size)
    local p = string.rep(".?", size)
    return string.match(s, p)
  end
  local m = f(80)
  assert(#m == 80)
  checkerror("too complex", f, 2000)
end
