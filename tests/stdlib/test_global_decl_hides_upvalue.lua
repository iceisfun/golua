-- Test: a `global X` declared in a nested function hides outer locals
-- What: name resolution stops at the first function that declares `global X`,
-- so X is the global there and in the functions nested in it, even when an
-- enclosing function has a local (or an inlined <const> local) named X. golua
-- used to look only at the function owning the local and resolved X to it as
-- an upvalue, so reads and writes went to the local, and assigning to a
-- <const> one failed to compile.

X = "global"
local X = "local"

assert((function() global X; return X end)() == "global")

;(function() global X; X = "set" end)()
assert(X == "local" and rawget(_G, "X") == "set")

-- Functions nested in the declaring one see the global too.
assert((function() global X; return (function() return X end)() end)() == "set")

-- Even after the function captured the local before the declaration.
do
  local a, b = (function() local a = X; global X; return a, X end)()
  assert(a == "local" and b == "set")
end

-- `global *` hides nothing.
assert((function() global *; return X end)() == "local")

-- A local declared after the global wins, in this function and inner ones.
assert((function() global X; local X = "inner"; return X end)() == "inner")
assert((function() global X; local X = "inner"; return (function() return X end)() end)() == "inner")

-- The declaration ends with its block.
assert((function() do global X end return X end)() == "local")

-- An inlined <const> local is hidden as well, and can be assigned through it.
do
  local K <const> = 1
  local f = assert(load("local K <const> = 1 return function() global K; K = 2 return K end"))()
  assert(f() == 2 and rawget(_G, "K") == 2)
  assert((function() global K; return K end)() == 2)
  assert(K == 1)
end

-- A global function declaration hides an outer local the same way.
do
  local G = "local"
  assert((function() global function G() return "fn" end return G() end)() == "fn")
  assert(G == "local" and type(rawget(_G, "G")) == "function")
end

X, K, G = nil, nil, nil
