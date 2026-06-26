-- Register reuse for chained index/field and and/or expressions must not
-- clobber a live local that a sibling sub-expression still reads. These are the
-- cases that broke when the optimization computed the left/table operand
-- directly into the destination register even when it was a live local.

-- and/or into a live local whose value the right operand reads:
do
  local function stack(n) n = ((n == 0) or stack(n - 1)) end
  stack(5)  -- must not raise "arithmetic on a boolean value (local 'n')"
  print("stack ok")
  --> =stack ok
end

-- indexed assignment whose key references the assigned local:
do
  local a = {x = {x = 42}}
  local x = "x"
  x = a[x][x]
  print(x)
  --> =42
end

-- and/or value semantics into a live local (right operand reads the local):
do
  local n = 3
  n = (n == 0) or (n + 10)
  print(n)
  --> =13
end

-- deep chains compile and evaluate (reuse one register, not one per level):
do
  print((load("return " .. ("1 and "):rep(300) .. "7"))())
  --> =7
  print((load("return " .. ("nil or "):rep(300) .. "9"))())
  --> =9
  -- a 300-deep index chain must COMPILE (load returns a function, not nil):
  print(type(load("local t = {}\nreturn t" .. ("[1]"):rep(300))))
  --> =function
end
