-- GETTABUP/SETTABUP errors should include upvalue name context

-- load with env=nil makes _ENV a nil upvalue
local f = load("return type(print)", "test", "t", nil)
local ok, err = pcall(f)
assert(not ok)
assert(err:find("%(upvalue '_ENV'%)"), "expected upvalue context in: " .. err)

-- SETTABUP case
local f2 = load("x = 1", "test2", "t", nil)
local ok2, err2 = pcall(f2)
assert(not ok2)
assert(err2:find("%(upvalue '_ENV'%)"), "expected upvalue context in: " .. err2)

print("PASS")
