-- Long constant expression chain should compile (constant folding)
local code = "return " .. string.rep("1+", 249) .. "1"
local f, err = load(code)
assert(f, "250 constant additions should compile, got: " .. tostring(err))
assert(f() == 250, "expected 250, got: " .. tostring(f()))

-- Even longer chain
local code2 = "return " .. string.rep("1+", 999) .. "1"
local f2, err2 = load(code2)
assert(f2, "1000 constant additions should compile, got: " .. tostring(err2))
assert(f2() == 1000, "expected 1000")

print("OK")
