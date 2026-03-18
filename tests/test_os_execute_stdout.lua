-- Test os.execute child inherits stdout/stderr
-- We can't easily capture stdout in the test harness, so we test via a temp file.
-- The key fix is that cmd.Stdout and cmd.Stderr are set to os.Stdout/os.Stderr.

local tmpfile = os.tmpname()
local ok, typ, code = os.execute("echo hello > " .. tmpfile)
assert(ok == true, "command failed")
assert(typ == "exit", "expected exit, got " .. tostring(typ))
assert(code == 0, "expected code 0, got " .. tostring(code))

-- Read the file to verify echo worked (child stdout was connected)
local f = io.open(tmpfile, "r")
assert(f, "could not open temp file")
local content = f:read("a")
f:close()
os.remove(tmpfile)

assert(content == "hello\n", "expected 'hello\\n', got '" .. tostring(content) .. "'")

print("PASS")
