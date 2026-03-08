-- Test merge_stderr option

-- exec.run with merge_stderr merges stderr into stdout
local result = exec.run("sh", "-c", "echo out; echo err >&2; echo out2", {merge_stderr = true})
assert(result.success == true, "should succeed")
assert(result.stderr == "", "stderr should be empty when merged, got: " .. tostring(result.stderr))
-- stdout should contain both stdout and stderr lines (order may vary due to buffering)
assert(string.find(result.stdout, "out"), "stdout should contain 'out', got: " .. tostring(result.stdout))
assert(string.find(result.stdout, "err"), "stdout should contain 'err', got: " .. tostring(result.stdout))
assert(string.find(result.stdout, "out2"), "stdout should contain 'out2', got: " .. tostring(result.stdout))

-- exec.run without merge_stderr keeps them separate
local result2 = exec.run("sh", "-c", "echo out; echo err >&2")
assert(result2.stdout == "out\n", "stdout should be 'out\\n', got: " .. tostring(result2.stdout))
assert(result2.stderr == "err\n", "stderr should be 'err\\n', got: " .. tostring(result2.stderr))

-- exec.spawn with merge_stderr
local p = exec.spawn("sh", "-c", "echo hello; echo world >&2", {merge_stderr = true})
local lines = {}
for line in p:readlines() do
    lines[#lines + 1] = line
end
p:wait()
-- Both lines should appear in stdout
local combined = table.concat(lines, "\n")
assert(string.find(combined, "hello"), "should contain 'hello', got: " .. combined)
assert(string.find(combined, "world"), "should contain 'world', got: " .. combined)

-- exec.run_shell with merge_stderr
local result3 = exec.run_shell("echo ok; echo fail >&2", {merge_stderr = true})
assert(result3.stderr == "", "stderr should be empty when merged")
assert(string.find(result3.stdout, "ok"), "stdout should contain 'ok'")
assert(string.find(result3.stdout, "fail"), "stdout should contain 'fail'")

-- Options table doesn't interfere when merge_stderr is false or absent
local result4 = exec.run("echo", "test", {merge_stderr = false})
assert(result4.stdout == "test\n", "should work with merge_stderr=false, got: " .. tostring(result4.stdout))

print("OK")
