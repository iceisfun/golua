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

print("PASS")
