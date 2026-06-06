-- Generic-for loop variables must resolve to the correct names via
-- debug.getlocal even when the loop body declares its own locals or nested
-- loops. Regression: the loop-variable visibility fixup computed its start
-- index after the body was compiled, so it grabbed still-active body locals
-- instead of the loop variables, swapping their debug names (e.g. reporting
-- the value of `k` under the name `v`).

-- Report the value debug.getlocal associates with a given variable name,
-- searching the caller's frame (level 2 from inside this helper).
local function valueOf(target)
  for i = 1, 16 do
    local n, v = debug.getlocal(2, i)
    if not n then break end
    if n == target then return v end
  end
  return "<missing>"
end

-- Body local before a nested numeric-for.
for k, v in pairs({ZZ = 99}) do
  local names = {}
  for j = 1, 1 do
    print("a:k=" .. tostring(valueOf("k")) .. " v=" .. tostring(valueOf("v")))
  end
  break
end
--> =a:k=ZZ v=99

-- Three loop variables with a preceding body local.
local function tri(_, i)
  i = i + 1
  if i <= 1 then return i, i * 10, i * 100 end
end
for a, b, c in tri, nil, 0 do
  local pad = true
  print("b:a=" .. tostring(valueOf("a"))
     .. " b=" .. tostring(valueOf("b"))
     .. " c=" .. tostring(valueOf("c")))
end
--> =b:a=1 b=10 c=100

-- Nested generic-for: inner loop vars must not shadow the outer ones' names.
for ok, ov in pairs({P = 1}) do
  for ik, iv in pairs({Q = 2}) do
    print("c:ik=" .. tostring(valueOf("ik")) .. " iv=" .. tostring(valueOf("iv")))
  end
end
--> =c:ik=Q iv=2
