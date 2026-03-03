-- Test: files.lua - os.date and os.time
-- From: files.lua
-- What: Tests date formatting, time conversion, date table normalization, and edge cases

do
  local function checkerr (msg, f, ...)
    local stat, err = pcall(f, ...)
    assert(not stat and string.find(err, msg, 1, true),
      "expected error with '" .. msg .. "', got: " .. tostring(err))
  end

  assert(os.date("") == "")
  assert(os.date("!") == "")
  assert(os.date("\0\0") == "\0\0")
  local t = os.time()
  local D = os.date("*t", t)
  assert(os.time(D) == t)

  checkerr("invalid conversion specifier", os.date, "%")
  checkerr("invalid conversion specifier", os.date, "%9")
  checkerr("not an integer", os.time, {year=1000, month=1, day=1, hour='x'})
  checkerr("missing", os.time, {hour = 12})

  -- basic date field checks
  assert(type(D.year) == "number")
  assert(type(D.month) == "number" and D.month >= 1 and D.month <= 12)
  assert(type(D.day) == "number" and D.day >= 1 and D.day <= 31)
  assert(type(D.hour) == "number" and D.hour >= 0 and D.hour <= 23)
  assert(type(D.min) == "number" and D.min >= 0 and D.min <= 59)
  assert(type(D.sec) == "number" and D.sec >= 0 and D.sec <= 60)
end
