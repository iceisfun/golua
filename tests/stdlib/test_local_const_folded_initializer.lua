-- Test: <const> locals whose initializer folds to a constant take no register
-- What: reference Lua makes a <const> local a compile-time constant whenever
-- its initializer is one once parsed (luaK_exp2const): a literal, another such
-- local, folded arithmetic over them, `not` of a constant, or an and/or that
-- reduces to a constant right operand. golua only accepted literals and
-- arithmetic, so the others kept a register and an upvalue slot, a chunk with
-- more than 200 of them failed with "too many local variables", and errors
-- named the variable where reference gives no name.

local function many(init)
  local src = {"local K <const> = 3", "local KS <const> = 'str'"}
  for i = 1, 250 do src[#src + 1] = ("local k%d <const> = %s"):format(i, init) end
  src[#src + 1] = "return k1, k250"
  local f, err = load(table.concat(src, "\n"))
  if not f then return nil, err end
  return f()
end

local cases = {
  {"K", 3}, {"(K)", 3}, {"-K", -3}, {"K * 2", 6}, {"KS", "str"},
  {"not nil", true}, {"not K", false}, {"not not nil", false},
  {"not 'x'", false}, {"not 0.0", false}, {"not (1 + 2)", false},
  {"false or 7", 7}, {"nil or K", 3}, {"K and 5", 5}, {"true and nil", nil},
  {"(nil or 'a')", "a"}, {"(K) and 'x'", "x"}, {"KS and 1", 1},
  {"(1 and 2) and 3", 3}, {"1 and (2 and 3)", 3},
  {"1.0 | 0", 1}, {"-(-9223372036854775807 - 1)", math.mininteger},
}
for _, c in ipairs(cases) do
  local a, b = many(c[1])
  assert(a == c[2] and b == c[2], c[1] .. ": " .. tostring(b))
end

-- An initializer that leaves a jump pending stays a variable, as in reference.
for _, init in ipairs{"nil and 1", "1 or 2", "K or x", "nil or 2 or 3",
                      "not (nil and 1)", "-0.0", "'a' .. 'b'"} do
  local ok, err = many(init)
  assert(not ok and err:find("too many local variables"), init)
end

-- Such a local is not captured as an upvalue and has no debug entry.
do
  local K <const> = 1
  local function upname(mk) return (debug.getupvalue(mk(), 1)) end
  assert(upname(function() local a <const> = K return function() return a end end) == nil)
  assert(upname(function() local a <const> = not nil return function() return a end end) == nil)
  assert(upname(function() local a <const> = K and "x" return function() return a end end) == nil)
  local A <const> = nil
  local B <const> = (A or 3)
  local D <const> = (not A)
  assert(B == 3 and D == true)
  local ok, msg = pcall(function() return #B end)
  assert(not ok and not msg:find("upvalue"), msg)
  for i = 1, 20 do
    local name = debug.getlocal(1, i)
    if not name then break end
    assert(name ~= "K" and name ~= "B" and name ~= "D", name)
  end
end

-- The left operands an and/or passes over still run, in order.
do
  local order = {}
  local function f(x) order[#order + 1] = x return nil end
  local K <const> = 1
  local a, b <const> = f(1), (f(2) or 1) and not (f(3) or K)
  assert(a == nil and b == false)
  local c <const> = (f(4) or "s") and K
  assert(c == 1)
  assert(table.concat(order, ",") == "1,2,3,4")
end
