-- goto error messages should include line number
local f, err = load("goto x")
assert(err:find(":1:"), "expected :1: in: " .. err)
assert(err:find("no visible label 'x'"), "expected no visible label in: " .. err)

-- Multi-line case
local f2, err2 = load("local a = 1\ngoto x")
assert(err2:find(":2:"), "expected :2: in: " .. err2)
