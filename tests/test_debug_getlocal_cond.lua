-- debug.getlocal returns correct values for locals inside conditional blocks.
-- Regression test: the compiler's compileCondJump was leaking the condition
-- register, causing locals in the then/body block to be allocated at a higher
-- register than their Locals table entry indicated.

-- Helper: find a local by name and verify its value
local function checkLocal(level, name, expected)
  level = level + 1 -- account for this helper frame
  for idx = 1, 50 do
    local n, v = debug.getlocal(level, idx)
    if n == nil then break end
    if n == name then
      assert(v == expected,
        string.format("getlocal %q: expected %s, got %s", name, tostring(expected), tostring(v)))
      return
    end
  end
  error(string.format("local %q not found", name))
end

-- Test 1: local inside if-with-comparison inside for loop
for i = 1, 1 do
  if i == 1 then
    local x = "hello"
    checkLocal(1, "x", "hello")
  end
end

-- Test 2: local inside if-true inside for loop
for i = 1, 1 do
  if true then
    local x = "world"
    checkLocal(1, "x", "world")
  end
end

-- Test 3: local inside while-with-condition
local cond = true
while cond do
  local x = "test"
  checkLocal(1, "x", "test")
  cond = false
end

-- Test 4: nested if blocks
for i = 1, 1 do
  if i == 1 then
    local a = 10
    if a == 10 then
      local b = 20
      checkLocal(1, "a", 10)
      checkLocal(1, "b", 20)
    end
  end
end

-- Test 5: elseif branch
for i = 1, 1 do
  if i == 99 then
    local x = "nope"
  elseif i == 1 then
    local y = "found"
    checkLocal(1, "y", "found")
  end
end

print("PASS")
