-- Test that compiler errors include source:line prefix

-- break outside loop
local ok, err = load("do break end\n")
assert(not ok, "break outside loop should fail")
-- Error should have [string "..."]:LINE: format
assert(err:find("^%[string .+%]:%d+:"), "break error should have source:line prefix: " .. err)

-- goto undefined label
ok, err = load("print('a')\nprint('b')\ngoto nowhere\nprint('c')\n")
assert(not ok, "goto undefined should fail")
assert(err:find("^%[string .+%]:%d+:"), "goto error should have source:line prefix: " .. err)

print("OK")
