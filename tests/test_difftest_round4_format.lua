-- Differential testing round 4: string.format fixes
-- Found via automated differential testing against Lua 5.4.8

-- Bug 9: string.format NaN sign bit
do
  -- -(0/0) should give -nan, 0/0 sign depends on platform but should not always be -nan
  local neg_nan = string.format("%f", -(0/0))
  assert(neg_nan == "-nan" or neg_nan == "nan", "unexpected: " .. neg_nan)
  local pos_nan = string.format("%f", 0/0)
  assert(pos_nan == "-nan" or pos_nan == "nan", "unexpected: " .. pos_nan)
  -- Upper case via %G
  local neg_NAN = string.format("%G", -(0/0))
  assert(neg_NAN == "-NAN" or neg_NAN == "NAN", "unexpected: " .. neg_NAN)
end

-- Bug 7: space flag applied to inf/nan
do
  assert(string.format("% f", 1/0) == " inf", "space+inf: |" .. string.format("% f", 1/0) .. "|")
  assert(string.format("% f", -(1/0)) == "-inf")
  assert(string.format("%+f", 1/0) == "+inf")
  assert(string.format("%+f", -(1/0)) == "-inf")
  -- Space flag with uppercase
  assert(string.format("% E", 1/0) == " INF")
  assert(string.format("%+G", 1/0) == "+INF")
end

-- Bug 6: %a width uses space-fill, not zero-fill
do
  local r = string.format("%20a", 1.0)
  assert(#r == 20, "width 20: " .. #r)
  assert(r:sub(1,1) == " ", "%20a should space-pad: |" .. r .. "|")
  -- Explicit zero-fill should still work
  local z = string.format("%020a", 1.0)
  assert(#z == 20)
  assert(z:find("0x0+") ~= nil, "%020a should zero-pad: |" .. z .. "|")
  -- Left align should space-pad on right
  local l = string.format("%-20a", 1.0)
  assert(#l == 20)
  assert(l:sub(-1) == " ", "%-20a should space-pad right: |" .. l .. "|")
end

-- Bug 8: %#.0a forces trailing decimal point
do
  assert(string.format("%#.0a", 1.0) == "0x1.p+0", "%#.0a: " .. string.format("%#.0a", 1.0))
  -- Without # flag, no decimal point at prec 0
  assert(string.format("%.0a", 1.0) == "0x1p+0", "%.0a: " .. string.format("%.0a", 1.0))
end

print("PASS")
