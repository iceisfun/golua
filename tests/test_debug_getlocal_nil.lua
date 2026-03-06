-- debug.getlocal/setlocal return nil (not nothing) for out-of-range indices

-- Function form: getlocal(func, index)
local function f(a, b) end
assert(select("#", debug.getlocal(f, 3)) == 1, "getlocal func form should return 1 value")
assert(debug.getlocal(f, 3) == nil)

-- Level form: getlocal(level, index)
local function test()
    local a = 1
    assert(select("#", debug.getlocal(1, 99)) == 1, "getlocal level form should return 1 value")
    assert(debug.getlocal(1, 99) == nil)
    assert(select("#", debug.setlocal(1, 99, "x")) == 1, "setlocal should return 1 value")
    assert(debug.setlocal(1, 99, "x") == nil)
end
test()

print("PASS")
