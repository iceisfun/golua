-- Test: line attribution for post-parse compile errors.
--
-- goto-into-scope is detected after the block is parsed, so its prefix line
-- is the resolution point (verified against lua5.5.0), while "break outside
-- loop" is anchored at the break statement's own line in Lua 5.5 (it was the
-- EOF line in Lua 5.4 — see test_break_outside_loop_error.lua).

-- goto scope error: reports the label-resolution line (4), matching lua5.5.0:
-- [string "goto skip"]:4: <goto skip> at line 1 jumps into the scope of 'x'
local f, err = load("goto skip\nlocal x = 1\n::skip::\nprint(x)")
assert(f == nil, "should fail to compile")
assert(err:find(":4:"), "expected line 4 in goto error, got: " .. err)

-- break outside loop (with trailing newline): Lua 5.5 anchors at the break's
-- own line (1), NOT the EOF line (2).
local f2, err2 = load("break\n")
assert(f2 == nil, "should fail")
assert(err2:find(":1: break outside loop near 'break'"),
  "expected break error at line 1, got: " .. err2)

-- break without trailing newline: still line 1.
local f3, err3 = load("break")
assert(f3 == nil, "should fail")
assert(err3:find(":1: break outside loop near 'break'"),
  "expected line 1 in break error, got: " .. err3)

print("OK")
