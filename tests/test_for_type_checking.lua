-- Test: nextvar.lua - Type checking in numeric for loops
-- From: nextvar.lua
-- What: Tests that numeric for loops preserve the correct type (float vs integer) of the control variable based on the initial value and step types.

do  -- checking types
  local c
  local function checkfloat (i)
    assert(math.type(i) == "float")
    c = c + 1
  end

  c = 0; for i = 1.0, 10 do checkfloat(i) end
  assert(c == 10)

  c = 0; for i = -1, -10, -1.0 do checkfloat(i) end
  assert(c == 10)

  local function checkint (i)
    assert(math.type(i) == "integer")
    c = c + 1
  end

  local m = math.maxinteger
  c = 0; for i = m, m - 10, -1 do checkint(i) end
  assert(c == 11)

  c = 0; for i = 1, 10.9 do checkint(i) end
  assert(c == 10)

  c = 0; for i = 10, 0.001, -1 do checkint(i) end
  assert(c == 10)

  c = 0; for i = 1, "10.8" do checkint(i) end
  assert(c == 10)

  c = 0; for i = 9, "3.4", -1 do checkint(i) end
  assert(c == 6)

  c = 0; for i = 0, " -3.4  ", -1 do checkint(i) end
  assert(c == 4)

  c = 0; for i = 100, "96.3", -2 do checkint(i) end
  assert(c == 2)

  c = 0; for i = 1, math.huge do if i > 10 then break end; checkint(i) end
  assert(c == 10)

  c = 0; for i = -1, -math.huge, -1 do
           if i < -10 then break end; checkint(i)
          end
  assert(c == 10)


  for i = math.mininteger, -10e100 do assert(false) end
  for i = math.maxinteger, 10e100, -1 do assert(false) end

end
