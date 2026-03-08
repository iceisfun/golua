-- Test exec.run basic functionality

-- exec module should exist
assert(type(exec) == "table", "exec module should be a table")
assert(type(exec.run) == "function", "exec.run should be a function")
assert(type(exec.spawn) == "function", "exec.spawn should be a function")
assert(type(exec.run_shell) == "function", "exec.run_shell should be a function")

-- Simple command execution
local result = exec.run("echo", "hello world")
assert(type(result) == "table", "exec.run should return a table")
assert(result.success == true, "echo should succeed")
assert(result.code == 0, "echo should exit with code 0")
assert(result.stdout == "hello world\n", "stdout should contain 'hello world\\n', got: " .. tostring(result.stdout))
assert(result.stderr == "", "stderr should be empty, got: " .. tostring(result.stderr))

-- Command with multiple arguments
local result2 = exec.run("echo", "a", "b", "c")
assert(result2.stdout == "a b c\n", "expected 'a b c\\n', got: " .. tostring(result2.stdout))

-- Command that fails
local ok, err = pcall(exec.run, "false")
if ok then
    -- false exits with code 1 but doesn't error
    assert(result ~= nil)
end
local result3 = exec.run("sh", "-c", "exit 42")
assert(result3.success == false, "exit 42 should not be success")
assert(result3.code == 42, "expected exit code 42, got: " .. tostring(result3.code))

-- Command with stderr output
local result4 = exec.run("sh", "-c", "echo error >&2")
assert(result4.stderr == "error\n", "expected stderr 'error\\n', got: " .. tostring(result4.stderr))

print("OK")
