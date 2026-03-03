-- Test: cstack.lua - Stack overflow in recursive gsub
-- From: cstack.lua
-- What: Tests stack overflow detection in recursive string.gsub calls

do
  local function checkerror(expected, f, ...)
    local ok, msg = pcall(f, ...)
    assert(not ok and type(msg) == "string" and string.find(msg, expected),
           "expected error '" .. expected .. "' got: " .. tostring(msg))
  end

  local count = 0
  local function foo ()
    count = count + 1
    string.gsub("a", ".", foo)
  end
  checkerror("stack overflow", foo)
end
