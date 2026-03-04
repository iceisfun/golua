-- Test: nextvar.lua - For step is zero error
-- From: nextvar.lua
-- What: Tests that numeric for loops with a zero step raise an error for both integer and float steps.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  checkerror("'for' step is zero", function ()
    for i = 1, 10, 0 do end
  end)

  checkerror("'for' step is zero", function ()
    for i = 1, -10, 0 do end
  end)

  checkerror("'for' step is zero", function ()
    for i = 1.0, -10, 0.0 do end
  end)
end
