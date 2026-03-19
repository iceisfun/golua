-- Test os.time year range validation
-- Lua 5.4: tm_year is a C int, so year-1900 must fit in int32 range

if string.packsize("i") == 4 then
  -- Year that underflows int32 when converted to tm_year
  local ok, err = pcall(os.time, {year = -(1 << 31) + 1899, month = 1, day = 1})
  assert(not ok, "should reject underflow year")
  assert(string.find(err, "out-of-bound", 1, true), "wrong error: " .. err)
  print("underflow year rejected: OK")

  -- Another underflow
  ok, err = pcall(os.time, {year = -(1 << 31), month = 1, day = 1})
  assert(not ok, "should reject underflow year 2")
  assert(string.find(err, "out-of-bound", 1, true), "wrong error: " .. err)
  print("underflow year 2 rejected: OK")

  if math.maxinteger > 2^31 then
    -- Year that overflows int32
    ok, err = pcall(os.time, {year = (1 << 31) + 1900, month = 1, day = 1})
    assert(not ok, "should reject overflow year")
    assert(string.find(err, "out-of-bound", 1, true), "wrong error: " .. err)
    print("overflow year rejected: OK")
  end
else
  print("skipping: int size is not 4")
end

-- Normal years should work fine
assert(os.time({year = 2000, month = 1, day = 1}))
assert(os.time({year = 1970, month = 1, day = 1}))
print("normal years: OK")
print("PASS")
