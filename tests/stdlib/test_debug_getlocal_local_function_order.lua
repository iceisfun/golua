-- Test: local names when a block starts with `local function`
-- What: a `local function f` is activated before its CLOSURE is emitted, so it
-- can start at the same pc as the locals declared just before it. golua sorted
-- the debug entries by start pc alone and kept the inner f, added first when
-- its scope closed, ahead of the outer locals. Every name in the frame was then
-- shifted: debug.getlocal paired names with the wrong values and runtime
-- errors named the wrong variable.

local function names(level)
  local out = {}
  for i = 1, 20 do
    local n = debug.getlocal(level + 1, i)
    if not n then break end
    out[#out + 1] = n
  end
  return table.concat(out, ",")
end

local function errmsg(f, ...)
  local ok, msg = pcall(f, ...)
  assert(not ok)
  return msg
end

do
  local seen
  local function t(p)
    local a, b = 1, 2
    do
      local function f() end
      seen = names(1)
      assert(select(2, debug.getlocal(1, 2)) == 1)
      return a.x
    end
  end
  assert(errmsg(t):find("local 'a'"))
  assert(seen:find("^p,a,b,f"), seen)
end

assert(errmsg(function(p) do local function f() end; return p.x end end):find("local 'p'"))
assert(errmsg(function() local a; repeat local function f() end; return a.x until true end):find("local 'a'"))
assert(errmsg(function() local a; while true do local function f() end; return a.x end end):find("local 'a'"))
assert(errmsg(function()
  local a = 1
  do local b = 2 do local function f() end end end
  local function g() end
  return a.x
end):find("local 'a'"))

-- The loop variable of a numeric for, then a local function in the body.
do
  local seen
  local msg = errmsg(function()
    for i = 1, 2 do
      local function f() end
      seen = names(1)
      return i.x
    end
  end)
  assert(msg:find("local 'i'"), msg)
  assert(seen:find("i,f"), seen)
end

-- debug.setlocal writes the variable it names.
do
  local function h()
    local a = 1
    do
      local function f() end
      assert(debug.setlocal(1, 1, 10) == "a")
      return a
    end
  end
  assert(h() == 10)
end
