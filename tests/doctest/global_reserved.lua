-- Test that 'global' is a reserved keyword in golua (Lua 5.5).
--
-- In Lua 5.4, 'global' is an ordinary identifier -- you can use it as a
-- variable name, table field key, or function name.  In Lua 5.5 (and golua),
-- 'global' is a keyword used for global-variable declarations.
--
-- NOTE: In reference Lua 5.5, 'global' is a *soft* keyword -- it can still
-- appear as a local variable name or table field.  golua currently treats it
-- as a hard reserved word, which is stricter.  These tests document the
-- CURRENT golua behavior.

----------------------------------------------------------------------
-- 1. 'global' cannot be used as a local variable name
----------------------------------------------------------------------
do
  local f, err = load("local global = 1")
  print(f == nil)
  --> =true
  print(type(err) == "string" and err:find("near 'global'") ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 2. 'global' cannot appear as a bare assignment target (parsed as decl)
----------------------------------------------------------------------
do
  local f, err = load("global = 1")
  print(f == nil)
  --> =true
  print(type(err) == "string" and err:find("near '='") ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 3. 'function global() end' is invalid (global is not a <name>)
----------------------------------------------------------------------
do
  local f, err = load("function global() end")
  print(f == nil)
  --> =true
  print(type(err) == "string" and err:find("near 'global'") ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 4. 'for global = ...' is invalid
----------------------------------------------------------------------
do
  local f, err = load("for global = 1, 3 do end")
  print(f == nil)
  --> =true
  print(type(err) == "string" and err:find("near 'global'") ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 5. 'global' as table field key is invalid in golua (stricter than 5.5)
----------------------------------------------------------------------
do
  local f, err = load("return {global = 1}")
  print(f == nil)
  --> =true
  print(type(err) == "string" and err:find("global") ~= nil)
  --> =true
end

----------------------------------------------------------------------
-- 6. The 'global' declaration syntax DOES work (fresh env to avoid leaks)
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
