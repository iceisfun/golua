-- Empty chunk name should produce [string ""] in errors
local f, err = load("bad code", "")
assert(f == nil)
assert(err:find('%[string ""%]'), 'expected [string ""] in: ' .. err)

-- nil chunk name defaults to source text
local f2, err2 = load("bad code", nil)
assert(f2 == nil)
assert(err2:find('%[string "bad code"%]'), 'expected [string "bad code"] in: ' .. err2)
