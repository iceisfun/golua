-- package.searchpath argument validation and coercion parity.

do
  local ok, err

  -- Lua 5.4 checks argument #2 (path) before #1 (name).
  ok, err = pcall(package.searchpath)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)

  ok, err = pcall(package.searchpath, "mod")
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)

  -- Required args: name/path accept strings and numeric coercion, reject tables.
  ok, err = pcall(package.searchpath, {}, "?.lua")
  assert(not ok)
  assert(string.find(err, "bad argument #1", 1, true), err)

  ok, err = pcall(package.searchpath, "mod", {})
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)

  -- Optional sep/rep: nil uses defaults; numbers are coerced to strings.
  local s, msg = package.searchpath(123, "?.lua", nil, nil)
  assert(s == nil and type(msg) == "string")

  s, msg = package.searchpath("a", "?.lua", 7, 8)
  assert(s == nil and type(msg) == "string")
  assert(string.find(msg, "no file '", 1, true) == 1, msg)

  s, msg = package.searchpath("a", "?.lua;?/init.lua", 7, 8)
  assert(s == nil and type(msg) == "string")
  assert(string.find(msg, "\n\tno file '", 1, true) ~= nil, msg)

  -- Optional sep/rep reject non-coercible values.
  ok, err = pcall(package.searchpath, "a", "?.lua", {})
  assert(not ok)
  assert(string.find(err, "bad argument #3", 1, true), err)

  ok, err = pcall(package.searchpath, "a", "?.lua", ".", {})
  assert(not ok)
  assert(string.find(err, "bad argument #4", 1, true), err)
end
