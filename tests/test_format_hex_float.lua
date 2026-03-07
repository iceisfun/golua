-- Test: string.format %a/%A (hex float)
-- From: strings.lua
-- What: Tests string.format("%a", ...) and %A hex float formatting for various numbers, verifying ISO C format requirements and round-tripping through tonumber.

do
  local function matchhexa (n)
    local s = string.format("%a", n)
    assert(string.find(s, "^%-?0x[1-9a-f]%.?[0-9a-f]*p[-+]?%d+$"))
    assert(tonumber(s) == n)
    s = string.format("%A", n)
    assert(string.find(s, "^%-?0X[1-9A-F]%.?[0-9A-F]*P[-+]?%d+$"))
    assert(tonumber(s) == n)
  end
  for _, n in ipairs{0.1, -0.1, 1/3, -1/3, 1e30, -1e30,
                     -45/247, 1, -1, 2, -2, 3e-20, -3e-20} do
    matchhexa(n)
  end

  assert(string.find(string.format("%A", 0.0), "^0X0%.?0*P%+?0$"))
  assert(string.find(string.format("%a", -0.0), "^%-0x0%.?0*p%+?0$"))

  assert(string.find(string.format("%a", 1/0), "^inf"))
  assert(string.find(string.format("%A", -1/0), "^%-INF"))
  assert(string.find(string.format("%a", 0/0), "^%-?nan"))
  assert(string.find(string.format("%a", -0.0), "^%-0x0"))

  if not pcall(string.format, "%.3a", 0) then

  else
    assert(string.find(string.format("%+.2A", 12), "^%+0X%x%.%x0P%+?%d$"))
    assert(string.find(string.format("%.4A", -12), "^%-0X%x%.%x000P%+?%d$"))
  end
end
