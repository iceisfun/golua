-- Test that os.date arg#2 error includes "got TYPE"
local ok, err = pcall(os.date, "%Y", "x")
assert(not ok)
assert(err:find("got string"), "missing 'got string' in: " .. tostring(err))

local ok2, err2 = pcall(os.date, "%Y", true)
assert(not ok2)
assert(err2:find("got boolean"), "missing 'got boolean' in: " .. tostring(err2))

local ok3, err3 = pcall(os.date, "%Y", {})
assert(not ok3)
assert(err3:find("got table"), "missing 'got table' in: " .. tostring(err3))
print("OK")
