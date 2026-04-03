-- Test that 'global' is a soft keyword in golua (Lua 5.5).
--
-- In Lua 5.5, 'global' is only special at statement start. It can still be
-- used as a local variable name, table field key, function parameter, etc.

----------------------------------------------------------------------
-- 1. 'global' can be used as a local variable name
----------------------------------------------------------------------
do
  local f, err = load("local global = 1; return global")
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =1
  end
end

----------------------------------------------------------------------
-- 2. 'global = 1' at statement start is a declaration, not assignment
----------------------------------------------------------------------
do
  local f, err = load("global = 1")
  print(f == nil)
  --> =true
  -- parser treats "global =" as a global declaration, expects name list
end

----------------------------------------------------------------------
-- 3. 'function global() end' is valid (global is a NAME)
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("function global() return 42 end; return global()", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =42
  end
end

----------------------------------------------------------------------
-- 4. 'for global = ...' is valid (global as loop variable name)
----------------------------------------------------------------------
do
  local f, err = load("local s = 0; for global = 1, 3 do s = s + global end; return s")
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =6
  end
end

----------------------------------------------------------------------
-- 5. 'global' as table field key is valid
----------------------------------------------------------------------
do
  local f, err = load("return {global = 1}")
  print(f ~= nil)
  --> =true
  if f then
    local t = f()
    print(t.global)
    --> =1
  end
end

----------------------------------------------------------------------
-- 6. The 'global' declaration syntax works (fresh env to avoid leaks)
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("global grx1; grx1 = 5; return grx1", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =5
  end
end

----------------------------------------------------------------------
-- 7. 'global' with <const> attribute works (fresh env)
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("global grx2 <const> = 42; return grx2", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =42
  end
end

----------------------------------------------------------------------
-- 8. Multiple global declarations (fresh env)
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("global gra, grb, grc; gra = 1; grb = 2; grc = 3; return gra + grb + grc", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =6
  end
end

----------------------------------------------------------------------
-- 9. global<const> (no space before <) also works (fresh env)
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("global<const> gry = 99; return gry", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =99
  end
end

----------------------------------------------------------------------
-- 10. 'global <const> *' wildcard syntax compiles
----------------------------------------------------------------------
do
  local f, err = load("global <const> *; return true")
  print(f ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 11. 'global' as function parameter
----------------------------------------------------------------------
do
  local f, err = load("local function f(global) return global end; return f(99)")
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =99
  end
end

----------------------------------------------------------------------
-- 12. 'global' as method name via dot syntax
----------------------------------------------------------------------
do
  local env = setmetatable({}, {__index = _G})
  local f, err = load("local t = {}; function t.global() return 7 end; return t.global()", nil, nil, env)
  print(f ~= nil)
  --> =true
  if f then
    print(f())
    --> =7
  end
end
