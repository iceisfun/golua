-- Test exec options: cwd, env, timeout

-- cwd option changes working directory
local result = exec.run("pwd", {cwd = "/tmp"})
assert(result.success == true, "pwd should succeed")
-- /tmp may be a symlink (e.g., /private/tmp on macOS), so check suffix
assert(string.find(result.stdout, "tmp"), "cwd should be /tmp, got: " .. tostring(result.stdout))

-- cwd with spawn
local p = exec.spawn("pwd", {cwd = "/tmp"})
local line = p:readline()
assert(string.find(line, "tmp"), "spawn cwd should be /tmp, got: " .. tostring(line))
p:wait()

-- cwd with run_shell
local result2 = exec.run_shell("pwd", {cwd = "/tmp"})
assert(string.find(result2.stdout, "tmp"), "run_shell cwd should be /tmp, got: " .. tostring(result2.stdout))

-- env option sets environment variables
local result3 = exec.run("sh", "-c", "echo $MY_TEST_VAR", {env = {MY_TEST_VAR = "hello123"}})
assert(result3.stdout == "hello123\n", "env var should be set, got: " .. tostring(result3.stdout))

-- env replaces entire environment (PATH not inherited)
local result4 = exec.run("sh", "-c", "echo ${NONEXISTENT_VAR:-empty}", {env = {NONEXISTENT_VAR = "found"}})
assert(result4.stdout == "found\n", "env should set var, got: " .. tostring(result4.stdout))

-- env with spawn
local p2 = exec.spawn("sh", "-c", "echo $FOO", {env = {FOO = "bar"}})
local line2 = p2:readline()
assert(line2 == "bar", "spawn env should work, got: " .. tostring(line2))
p2:wait()

-- timeout option kills long-running process
local result5 = exec.run("sleep", "60", {timeout = 200})
assert(result5.success == false, "timed out process should not succeed")

-- timeout with run_shell
local result6 = exec.run_shell("sleep 60", {timeout = 200})
assert(result6.success == false, "timed out shell should not succeed")

-- timeout with spawn (process auto-killed on timeout)
local p3 = exec.spawn("sleep", "60", {timeout = 200})
local r3 = p3:wait()
assert(r3.success == false, "spawned process with timeout should be killed")

-- combining options
local result7 = exec.run("sh", "-c", "echo $COMBO; pwd; echo err >&2", {
    cwd = "/tmp",
    env = {COMBO = "works"},
    merge_stderr = true
})
assert(result7.success == true)
assert(string.find(result7.stdout, "works"), "combined: env should work, got: " .. tostring(result7.stdout))
assert(string.find(result7.stdout, "tmp"), "combined: cwd should work, got: " .. tostring(result7.stdout))
assert(string.find(result7.stdout, "err"), "combined: merge_stderr should work, got: " .. tostring(result7.stdout))
assert(result7.stderr == "", "combined: stderr should be empty when merged")

print("OK")
