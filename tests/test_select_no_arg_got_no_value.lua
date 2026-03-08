-- select() with no arguments should report "got no value" not "got nil"
local ok, err = pcall(select)
assert(err == "bad argument #1 to 'select' (number expected, got no value)",
  "expected 'got no value', got: " .. tostring(err))
print("OK")
