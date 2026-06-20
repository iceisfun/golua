-- Test: errors.lua - Too many local variables error
-- From: errors.lua
-- What: Tests compiler limits on local declarations.

-- 301 locals declared in a single statement. Like the reference compiler,
-- golua reserves all the registers (luaK_reserveregs) before adjustlocalvars
-- checks MAXVARS, so the register limit (255) is reported first.
do
  local s = "\nfunction foo ()\n  local "
  for i = 1, 300 do
    s = s .. "a" .. i .. ", "
  end
  s = s .. "a = 1\nend"
  local a, b = load(s)
  assert(string.find(b, "too many registers"))
end

-- 201 locals across separate statements stays within the register file, so the
-- MAXVARS (200) limit is the one reported.
do
  local s = "function foo ()\n"
  for i = 1, 201 do
    s = s .. "  local a" .. i .. " = 1\n"
  end
  s = s .. "end"
  local a, b = load(s)
  assert(string.find(b, "too many local variables"))
end
