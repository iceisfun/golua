-- Test compiler limits enforced by load()

-- MaxVars: 201 locals should fail
local code = {}
for i = 1, 201 do code[#code+1] = "local v"..i.." = "..i end
code[#code+1] = "return v1"
local f, err = load(table.concat(code, "\n"))
assert(f == nil, "expected compile error for 201 locals")
assert(string.find(err, "too many local"), "expected 'too many local' in: " .. err)

-- MaxVars: 200 locals (at limit) should work
code = {}
for i = 1, 200 do code[#code+1] = "local v"..i.." = "..i end
code[#code+1] = "return v1"
f, err = load(table.concat(code, "\n"))
assert(f ~= nil, "200 locals should compile, got error: " .. tostring(err))
assert(f() == 1, "expected 200-local function to return 1")

-- MaxVars: scoped locals don't accumulate across scopes
-- Each do..end block closes its locals, so we can reuse registers.
code = {}
for i = 1, 10 do
    code[#code+1] = "do"
    for j = 1, 20 do
        code[#code+1] = "  local v"..((i-1)*20+j).." = "..j
    end
    code[#code+1] = "end"
end
code[#code+1] = "return true"
f, err = load(table.concat(code, "\n"))
assert(f ~= nil, "scoped locals should compile, got error: " .. tostring(err))
assert(f() == true)

-- MaxRegs: function call with too many args should fail
-- f(1,1,...,1) with 261 args needs 262 registers (func + 261 args),
-- which exceeds MaxRegs=249.
local s = "local function f(...) end\nf(" .. string.rep("1,", 260) .. "1)"
f, err = load(s)
assert(f == nil, "expected compile error for 261-arg call")
assert(string.find(err, "too many registers") or string.find(err, "too many"),
    "expected 'too many registers' in: " .. tostring(err))

-- MaxRegs: method call with too many args should also fail
s = "local t = {}\nfunction t:f(...) end\nt:f(" .. string.rep("1,", 259) .. "1)"
f, err = load(s)
assert(f == nil, "expected compile error for 260-arg method call")
assert(string.find(err, "too many registers") or string.find(err, "too many"),
    "expected 'too many registers' in: " .. tostring(err))

-- MaxRegs: call at limit should work (247 args + func + local = 249 regs)
s = "local function f(...) end\nf(" .. string.rep("1,", 246) .. "1)"
f, err = load(s)
assert(f ~= nil, "247-arg call should compile, got error: " .. tostring(err))

print("PASS")
