-- Test: vm.top management for native function calls
-- Verifies that stack register positions are not corrupted when:
-- 1. pcall/ProtectedCall is used with table.concat or other optional-arg natives
-- 2. Nested __tostring calls recurse through ProtectedCall
-- 3. Inline nested function calls use variable results (c=0 CALL + SETLIST)
-- 4. Tail calls to native functions clear stale register data

-- Helper: tree node with recursive __tostring
local function make_node(name, children)
    return setmetatable({name = name, children = children or {}}, {
        __tostring = function(self)
            local parts = {self.name}
            for _, child in ipairs(self.children) do
                parts[#parts + 1] = tostring(child)
            end
            return table.concat(parts, ".")
        end
    })
end

-- 1. pcall + table.concat: stale register not seen as optional arg
local function test_pcall_concat()
    local ok = pcall(function() return 1 end)
    return table.concat({"a", "b"}, ".")
end
assert(test_pcall_concat() == "a.b", "pcall + concat failed")

-- 2. tostring(__tostring) + table.concat in same function
local function test_tostring_then_concat()
    local leaf = setmetatable({}, {
        __tostring = function() return "x" end
    })
    local s = tostring(leaf)
    return table.concat({"hello", "world"}, ".")
end
assert(test_tostring_then_concat() == "hello.world", "tostring then concat failed")

-- 3. Leaf node __tostring
local leaf = make_node("x")
assert(tostring(leaf) == "x", "leaf failed: " .. tostring(tostring(leaf)))

-- 4. One-level nesting
local parent = make_node("a", {make_node("b")})
assert(tostring(parent) == "a.b", "one-level failed: " .. tostring(tostring(parent)))

-- 5. Two children at same level
local two = make_node("a", {make_node("b"), make_node("c")})
assert(tostring(two) == "a.b.c", "two children failed: " .. tostring(tostring(two)))

-- 6. Depth-2 nesting (inline creation - tests c=0 CALL + SETLIST)
local deep = make_node("a", {make_node("b", {make_node("d")})})
assert(tostring(deep) == "a.b.d", "depth-2 failed: " .. tostring(tostring(deep)))

-- 7. Full tree after prior tostring calls (tests vm.top restoration across calls)
local tree = make_node("a", {
    make_node("b", {make_node("d"), make_node("e")}),
    make_node("c")
})
assert(tostring(tree) == "a.b.d.e.c", "full tree failed: " .. tostring(tostring(tree)))

-- 8. Separate node creation (tests that vm.top isn't corrupted by inline creation)
local d2 = make_node("d2")
local b2 = make_node("b2", {d2})
local a2 = make_node("a2", {b2})
assert(tostring(a2) == "a2.b2.d2", "separate creation failed")

-- 9. tostring called multiple times on same object (no accumulation of corruption)
for i = 1, 5 do
    assert(tostring(tree) == "a.b.d.e.c", "repeated tostring #" .. i .. " failed")
end

-- 10. pcall wrapping tostring of recursive structure
local ok, result = pcall(tostring, tree)
assert(ok, "pcall(tostring, tree) failed: " .. tostring(result))
assert(result == "a.b.d.e.c", "pcall tostring result wrong")

-- 11. string.format after pcall (another optional-arg native)
local function test_format_after_pcall()
    pcall(function() end)
    return string.format("%s-%s", "a", "b")
end
assert(test_format_after_pcall() == "a-b", "format after pcall failed")

-- 12. table.concat with explicit i,j after pcall
local function test_concat_ij_after_pcall()
    pcall(function() return 1 end)
    return table.concat({"a", "b", "c", "d"}, ".", 2, 3)
end
assert(test_concat_ij_after_pcall() == "b.c", "concat with i,j after pcall failed")

-- 13. Nested pcall (ProtectedCall calling ProtectedCall)
local function test_nested_pcall()
    local ok1, r1 = pcall(function()
        local ok2, r2 = pcall(function()
            return table.concat({"x", "y"}, "-")
        end)
        assert(ok2)
        return r2 .. "!" .. table.concat({"a", "b"}, ".")
    end)
    assert(ok1)
    return r1
end
assert(test_nested_pcall() == "x-y!a.b", "nested pcall failed")

-- 14. Zero-arg pcall (tests argc=0 correctly handled)
local ok_zero = pcall(function() end)
assert(ok_zero, "zero-arg pcall body failed")

-- 15. Deep recursion stress test (5 levels)
local deep5 = make_node("1", {
    make_node("2", {
        make_node("3", {
            make_node("4", {
                make_node("5")
            })
        })
    })
})
assert(tostring(deep5) == "1.2.3.4.5", "5-level deep tree failed")
