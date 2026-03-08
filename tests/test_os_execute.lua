-- Test os.execute

-- os.execute() with no args returns true (shell is available)
assert(os.execute() == true)
assert(os.execute(nil) == true)

-- os.execute with a successful command
local ok, typ, code = os.execute("true")
assert(ok == true, "expected ok=true, got " .. tostring(ok))
assert(typ == "exit", "expected type='exit', got " .. tostring(typ))
assert(code == 0, "expected code=0, got " .. tostring(code))

-- os.execute with a failing command
local ok2, typ2, code2 = os.execute("false")
assert(ok2 == nil, "expected ok=nil, got " .. tostring(ok2))
assert(typ2 == "exit", "expected type='exit', got " .. tostring(typ2))
assert(code2 ~= 0, "expected non-zero exit code")

-- os.execute with a command that exits with specific code
local ok3, typ3, code3 = os.execute("exit 42")
assert(ok3 == nil, "expected ok=nil for exit 42")
assert(typ3 == "exit")
assert(code3 == 42, "expected code=42, got " .. tostring(code3))

-- os.execute with echo (test command output goes somewhere)
local ok4, typ4, code4 = os.execute("echo hello >/dev/null")
assert(ok4 == true)
assert(typ4 == "exit")
assert(code4 == 0)

print("PASS")
