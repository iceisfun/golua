-- Test: math.lua - Helper functions (checkerror, eq, eqT)
-- From: math.lua
-- What: Defines helper functions used throughout the test file: checkerror for expected errors, eq for approximate float comparison, eqT for type-aware equality.

do
  -- number of bits in the mantissa of a floating-point number
  local floatbits = 24
  do
    local p = 2.0^floatbits
    while p < p + 1.0 do
      p = p * 2.0
      floatbits = floatbits + 1
    end
  end

  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local msgf2i = "number.* has no integer representation"

  -- float equality
  local function eq (a,b,limit)
    if not limit then
      if floatbits >= 50 then limit = 1E-11
      else limit = 1E-5
      end
    end
    -- a == b needed for +inf/-inf
    return a == b or math.abs(a-b) <= limit
  end


  -- equality with types
  local function eqT (a,b)
    return a == b and math.type(a) == math.type(b)
  end

  -- Verify the helpers work
  assert(eq(1.0, 1.0))
  assert(eq(1.0, 1.0 + 1e-15))
  assert(eqT(1, 1))
  assert(not eqT(1, 1.0))  -- same value, different type
  checkerror("attempt to call", function() local x = 1; x() end)

end
