-- Test exec.spawn and process object methods

-- Spawn a simple process and read output
local p = exec.spawn("echo", "hello from spawn")
assert(type(p) == "userdata", "spawn should return userdata, got: " .. type(p))

local line = p:readline()
assert(line == "hello from spawn", "expected 'hello from spawn', got: " .. tostring(line))

local result = p:wait()
assert(result.success == true, "echo should succeed")
assert(result.code == 0, "echo should exit with 0")

-- Test readlines iterator
local p2 = exec.spawn("sh", "-c", "echo line1; echo line2; echo line3")
local lines = {}
for line in p2:readlines() do
    lines[#lines + 1] = line
end
assert(#lines == 3, "expected 3 lines, got: " .. #lines)
assert(lines[1] == "line1", "expected 'line1', got: " .. tostring(lines[1]))
assert(lines[2] == "line2", "expected 'line2', got: " .. tostring(lines[2]))
assert(lines[3] == "line3", "expected 'line3', got: " .. tostring(lines[3]))
p2:wait()

-- Test is_complete
local p3 = exec.spawn("sleep", "0.1")
-- Should not be complete immediately (race-free: just check the method works)
assert(type(p3:is_complete()) == "boolean", "is_complete should return boolean")
p3:wait()
assert(p3:is_complete() == true, "should be complete after wait")

-- Test exit_code
local p4 = exec.spawn("sh", "-c", "exit 7")
p4:wait()
assert(p4:exit_code() == 7, "expected exit code 7, got: " .. tostring(p4:exit_code()))

-- Test write and close_stdin
local p5 = exec.spawn("sort")
p5:write("banana\n")
p5:write("apple\n")
p5:write("cherry\n")
p5:close_stdin()

local sorted = {}
for line in p5:readlines() do
    sorted[#sorted + 1] = line
end
p5:wait()
assert(sorted[1] == "apple", "expected 'apple', got: " .. tostring(sorted[1]))
assert(sorted[2] == "banana", "expected 'banana', got: " .. tostring(sorted[2]))
assert(sorted[3] == "cherry", "expected 'cherry', got: " .. tostring(sorted[3]))

-- Test stderr reading
local p6 = exec.spawn("sh", "-c", "echo err_msg >&2")
local errline = p6:stderr()
assert(errline == "err_msg", "expected stderr 'err_msg', got: " .. tostring(errline))
p6:wait()

-- Test kill
local p7 = exec.spawn("sleep", "60")
p7:kill()
local r7 = p7:wait()
assert(r7.success == false, "killed process should not succeed")

-- Test tostring
local p8 = exec.spawn("true")
local s = tostring(p8)
assert(string.find(s, "process:"), "tostring should contain 'process:', got: " .. s)
p8:wait()

print("OK")
