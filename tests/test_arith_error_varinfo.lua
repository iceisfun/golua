-- Arithmetic/bitwise error messages must include variable name context
-- Bug: GoLua only included varInfo for add/unm/bnot, not sub/mul/div/etc.

-- Helper to extract the parenthesized context from an error message
local function ctx(err)
    return err:match("%((.-)%)") or "MISSING"
end

-- sub
do
    local t = {}
    local ok, err = pcall(function() return t - 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "sub: expected local 't', got " .. ctx(err))
end

-- mul
do
    local t = {}
    local ok, err = pcall(function() return t * 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "mul: expected local 't', got " .. ctx(err))
end

-- div
do
    local t = {}
    local ok, err = pcall(function() return t / 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "div: expected local 't', got " .. ctx(err))
end

-- idiv
do
    local t = {}
    local ok, err = pcall(function() return t // 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "idiv: expected local 't', got " .. ctx(err))
end

-- mod
do
    local t = {}
    local ok, err = pcall(function() return t % 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "mod: expected local 't', got " .. ctx(err))
end

-- pow
do
    local t = {}
    local ok, err = pcall(function() return t ^ 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "pow: expected local 't', got " .. ctx(err))
end

-- band
do
    local t = {}
    local ok, err = pcall(function() return t & 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "band: expected local 't', got " .. ctx(err))
end

-- bor
do
    local t = {}
    local ok, err = pcall(function() return t | 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "bor: expected local 't', got " .. ctx(err))
end

-- bxor
do
    local t = {}
    local ok, err = pcall(function() return t ~ 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "bxor: expected local 't', got " .. ctx(err))
end

-- shl
do
    local t = {}
    local ok, err = pcall(function() return t << 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "shl: expected local 't', got " .. ctx(err))
end

-- shr
do
    local t = {}
    local ok, err = pcall(function() return t >> 1 end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "shr: expected local 't', got " .. ctx(err))
end

-- Right operand context
do
    local t = {}
    local ok, err = pcall(function() return 1 - t end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "right sub: expected local 't', got " .. ctx(err))
end

do
    local t = {}
    local ok, err = pcall(function() return 1 & t end)
    assert(not ok)
    assert(ctx(err) == "upvalue 't'", "right band: expected local 't', got " .. ctx(err))
end

