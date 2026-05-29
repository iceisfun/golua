-- Regression: "break outside loop" must be anchored at the break statement's
-- own line with "near 'break'" wording (Lua 5.5), not the EOF line with the
-- old "at line N" wording (Lua 5.4).

local function compileErr(src)
  local f, e = load(src, "=chunk")
  assert(f == nil, "expected compile error for: " .. src)
  return e
end

-- Bare break at top level -> line 1, near 'break'
assert(compileErr("break") == "chunk:1: break outside loop near 'break'",
  "bare break: " .. tostring(compileErr("break")))

-- Break on a later line inside a do-block -> that line, not EOF
local e2 = compileErr("do\n break\n x=1\nend")
assert(e2 == "chunk:2: break outside loop near 'break'", "do-block break: " .. tostring(e2))

-- Break inside a function body -> the break's line
local e3 = compileErr("function f()\n break\nend")
assert(e3 == "chunk:2: break outside loop near 'break'", "fn break: " .. tostring(e3))

print("PASS")
