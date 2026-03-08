-- Test: debug.getlocal returns correct values for locals in
-- tail-called pcall/xpcall frames.
-- When pcall/xpcall is called as a tail call (return pcall(...)),
-- debug.getlocal at the level above should still see the caller's locals.

-- Test 1: tail-called pcall
local function outer_pcall()
  local my_outer = "outer_val"
  return pcall(function()
    local name, val = debug.getlocal(3, 1)
    assert(name == "my_outer", "expected name 'my_outer', got: " .. tostring(name))
    assert(val == "outer_val", "expected val 'outer_val', got: " .. tostring(val))
  end)
end

local ok, err = outer_pcall()
assert(ok, "pcall failed: " .. tostring(err))
print("PASS: tail-called pcall getlocal")

-- Test 2: tail-called xpcall
local function outer_xpcall()
  local my_x = 42
  return xpcall(function()
    local name, val = debug.getlocal(3, 1)
    assert(name == "my_x", "expected name 'my_x', got: " .. tostring(name))
    assert(val == 42, "expected val 42, got: " .. tostring(val))
  end, function(e) return e end)
end

local ok2, err2 = outer_xpcall()
assert(ok2, "xpcall failed: " .. tostring(err2))
print("PASS: tail-called xpcall getlocal")

-- Test 3: non-tail pcall should still work (regression guard)
local function outer_nontail()
  local my_nt = "nontail"
  local ok3, err3 = pcall(function()
    local name, val = debug.getlocal(3, 1)
    assert(name == "my_nt", "expected name 'my_nt', got: " .. tostring(name))
    assert(val == "nontail", "expected val 'nontail', got: " .. tostring(val))
  end)
  assert(ok3, "non-tail pcall failed: " .. tostring(err3))
end

outer_nontail()
print("PASS: non-tail pcall getlocal")

-- Test 4: multiple locals
local function outer_multi()
  local a = "alpha"
  local b = "beta"
  local c = "gamma"
  return pcall(function()
    local n1, v1 = debug.getlocal(3, 1)
    local n2, v2 = debug.getlocal(3, 2)
    local n3, v3 = debug.getlocal(3, 3)
    assert(n1 == "a" and v1 == "alpha", "local 1: " .. tostring(n1) .. "=" .. tostring(v1))
    assert(n2 == "b" and v2 == "beta", "local 2: " .. tostring(n2) .. "=" .. tostring(v2))
    assert(n3 == "c" and v3 == "gamma", "local 3: " .. tostring(n3) .. "=" .. tostring(v3))
  end)
end

local ok4, err4 = outer_multi()
assert(ok4, "multi-local pcall failed: " .. tostring(err4))
print("PASS: tail-called pcall multiple locals")

print("ALL PASSED")
