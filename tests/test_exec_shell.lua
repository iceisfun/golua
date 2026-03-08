-- Test exec.run_shell

-- Basic shell command
local result = exec.run_shell("echo hello")
assert(result.success == true, "echo should succeed")
assert(result.stdout == "hello\n", "expected 'hello\\n', got: " .. tostring(result.stdout))

-- Shell features (pipes, redirects)
local result2 = exec.run_shell("echo 'abc' | tr 'a-z' 'A-Z'")
assert(result2.stdout == "ABC\n", "expected 'ABC\\n', got: " .. tostring(result2.stdout))

-- Shell variable expansion
local result3 = exec.run_shell("echo $((2 + 3))")
assert(result3.stdout == "5\n", "expected '5\\n', got: " .. tostring(result3.stdout))

-- Failure
local result4 = exec.run_shell("exit 1")
assert(result4.success == false)
assert(result4.code == 1)

print("OK")
