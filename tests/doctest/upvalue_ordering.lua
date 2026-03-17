-- Upvalue registration order must match Lua 5.4's source order.
-- LHS targets are processed before RHS, so upvalues first referenced
-- in assignment targets get lower indices than those only in the RHS.

-- Test 1: reversed assignment targets register upvalues in LHS order
do
    local a, b, c = 1, 2, 3
    local function f() c, a, b = a, b, c end
    local n1 = debug.getupvalue(f, 1)
    local n2 = debug.getupvalue(f, 2)
    local n3 = debug.getupvalue(f, 3)
    print(n1, n2, n3)
    --> =c	a	b
end

-- Test 2: mixed plain + indexed targets
do
    local a, b, c = 1, 2, 3
    local function f() a, b[c], c = 10, 20, 30 end
    local n1 = debug.getupvalue(f, 1)
    local n2 = debug.getupvalue(f, 2)
    local n3 = debug.getupvalue(f, 3)
    print(n1, n2, n3)
    --> =a	b	c
end

-- Test 3: debug.upvaluejoin uses correct indices
do
    local a, b, c = 1, 2, 3
    local function f() c, a, b = a, b, c end
    local x, y, z = 10, 20, 30
    local function g() return x, y, z end
    -- f's upvalue 1 is 'c', g's upvalue 1 is 'x'
    -- After join, f's 'c' upvalue shares g's 'x' upvalue
    debug.upvaluejoin(f, 1, g, 1)
    f()
    print(a, b, c)
    --> =2	10	3
end
