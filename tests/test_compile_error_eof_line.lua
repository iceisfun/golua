-- Test: compile errors use EOF line for prefix (matching Lua 5.4)

-- goto scope error: should report last line of chunk
local f, err = load("goto skip\nlocal x = 1\n::skip::\nprint(x)")
assert(f == nil, "should fail to compile")
-- Lua 5.5: [string "goto skip"]:4: <goto skip> at line 1 jumps into the scope of 'x'
assert(err:find(":4:"), "expected line 4 (EOF line) in error, got: " .. err)

-- break outside loop: "break\n" has EOF at line 2
local f2, err2 = load("break\n")
assert(f2 == nil, "should fail")
assert(err2:find(":2:"), "expected line 2 (EOF line) in error, got: " .. err2)

-- break without trailing newline: EOF at line 1
local f3, err3 = load("break")
assert(f3 == nil, "should fail")
assert(err3:find(":1:"), "expected line 1 in error, got: " .. err3)

print("OK")
