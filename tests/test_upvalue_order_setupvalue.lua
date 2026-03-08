-- Bug: Upvalue ordering bug causes debug.setupvalue to modify wrong variable
-- When upvalues are first encountered as multi-assignment targets,
-- their indices are reversed. This means debug.setupvalue(f, 1, val)
-- modifies the wrong upvalue.

local a = "A"
local b = "B"

local function setter() a, b = "X", "Y" end
local function getter() return a, b end

-- Setting upvalue 1 should modify 'a' (first variable in the multi-assign)
debug.setupvalue(setter, 1, "FIRST")
local va, vb = getter()
assert(va == "FIRST", "expected a='FIRST' after setupvalue(setter, 1, 'FIRST'), got a='" .. tostring(va) .. "'")
assert(vb == "B", "expected b='B' (unchanged), got b='" .. tostring(vb) .. "'")

print("PASSED")
