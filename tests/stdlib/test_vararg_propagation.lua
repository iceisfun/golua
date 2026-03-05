-- Test: Variadic argument propagation, select, and tail-call integrity

do
  -- select(n, ...) after forwarding through two layers of vararg passing
  local function f(...)
    return select(3, ...)
  end
  local function g(...)
    return f(...)
  end
  assert(g(1, 2, 3, 4) == 3)

  -- Tail call preserves full vararg return list
  local function passthrough(...)
    return ...
  end
  local function tail(...)
    return passthrough(...)
  end
  local a, b, c = tail(1, 2, 3)
  assert(a == 1 and b == 2 and c == 3)

  print("PASS: vararg propagation")
end
