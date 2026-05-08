-- Mixed list/keyed constructors under Lua 5.5 # semantics: # walks
-- forward from index 1 returning the first contiguous run.

do
  -- Array part contiguous despite having keyed entries: 5.4 and 5.5 agree.
  local t0 = {{}, [3] = {}, [2] = {}}
  assert(#t0 == 3, "expected #t0 == 3, got " .. tostring(#t0))

  -- 5.5 first-hole semantics: positional fields with explicit nil produce
  -- t[2]=false, t[3]=nil-or-keyed; walking from t[1] stops at the first nil.
  local t = {"a", false, [4] = {}, [5] = nil, [3] = {}}
  t[2] = nil
  assert(#t == 1, "expected #t == 1, got " .. tostring(#t))

  -- table.remove with position past #t errors with "position out of bounds".
  local ok, err = pcall(table.remove, t, -1)
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "position out of bounds", 1, true), err)
end

-- Constructor with explicit interior nil: # stops at the first hole.
do
  local t2 = {1, false, 1, nil, "a", nil, [4] = "x"}
  -- Under 5.5, #t2 == 3 because t[4] is nil (positional nil wins over keyed).
  assert(#t2 == 3, "expected #t2 == 3, got " .. tostring(#t2))

  -- Removing past #t2 + 1 errors.
  local ok, err = pcall(table.remove, t2, 5)
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "position out of bounds", 1, true), err)
end
