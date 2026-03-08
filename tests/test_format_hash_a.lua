local function eq(actual, expected, label)
  assert(actual == expected,
    label .. " expected [" .. expected .. "] got [" .. actual .. "]")
end

local exactCases = {
  {"%a", 1.0, "0x1p+0"},
  {"%#a", 1.0, "0x1.p+0"},
  {"%#a", 2.0, "0x1.p+1"},
  {"%#a", 0.5, "0x1.p-1"},
  {"%#A", 1.0, "0X1.P+0"},
  {"%#.0a", 1.0, "0x1.p+0"},
  {"%#.0A", 1.0, "0X1.P+0"},
  {"%#.0a", 0.0, "0x0.p+0"},
  {"%#.0a", -0.0, "-0x0.p+0"},
  {"%#.2a", 1.0, "0x1.00p+0"},
  {"%#.0a", 1.5, "0x2.p+0"},
  {"%#.0a", 1e-308, "0x0.p-1022"},
  {"%#.0A", 1e-308, "0X0.P-1022"},
  {"%#.0a", 5e-324, "0x0.p-1022"},
  {"%#.0A", 5e-324, "0X0.P-1022"},
  {"%#.0a", 2e-308, "0x1.p-1022"},
  {"%#.0A", 2e-308, "0X1.P-1022"},
  {"%.0a", 1e-308, "0x0p-1022"},
  {"%.0A", 1e-308, "0X0P-1022"},
  {"%#.0a", -1e-308, "-0x0.p-1022"},
  {"%#.0a", -5e-324, "-0x0.p-1022"},
  {"%#.1a", 1e-308, "0x0.7p-1022"},
  {"%#.2a", 5e-324, "0x0.00p-1022"},
  {"%#.0a", 2.225073858507201e-308, "0x1.p-1022"},
  {"%#+.0a", 1.0, "+0x1.p+0"},
  {"% #.0a", 1.0, " 0x1.p+0"},
  {"%#20.0a", 1e-308, "          0x0.p-1022"},
  {"%#-20.0a", 1e-308, "0x0.p-1022          "},
  {"%#+20.0a", 1e-308, "         +0x0.p-1022"},
}

for _, c in ipairs(exactCases) do
  eq(string.format(c[1], c[2]), c[3], c[1] .. " " .. tostring(c[2]))
end

eq(string.format("#%010.0a#", 1.0), "#0x00001p+0#", "zero-pad without #")
eq(string.format("#%#010.0a#", 1.0), "#0x0001.p+0#", "zero-pad with #")
eq(string.format("#%#-10.0a#", 1.0), "#0x1.p+0   #", "left-align with #")

for _, n in ipairs({1.0, 2.0, 0.5, 1.5, 1e-308, 5e-324, 2e-308, -1e-308, -5e-324}) do
  local shortest = string.format("%a", n)
  local shortestAlt = string.format("%#a", n)
  local plain = string.format("%.0a", n)
  local alt = string.format("%#.0a", n)
  local plainUpper = string.format("%.0A", n)
  local altUpper = string.format("%#.0A", n)

  assert(not plain:find("%.", 1, false), "plain .0a should have no decimal point: " .. plain)
  assert(not plainUpper:find("%.", 1, false), "plain .0A should have no decimal point: " .. plainUpper)
  assert(alt:find("%.", 1, false), "alternate .0a should force decimal point: " .. alt)
  assert(altUpper:find("%.", 1, false), "alternate .0A should force decimal point: " .. altUpper)
  assert(tonumber(shortest) == n, "shortest %a should round-trip: " .. shortest)
  assert(tonumber(shortestAlt) == n, "shortest %#a should round-trip: " .. shortestAlt)
  assert(tonumber(plain) == tonumber(alt), "# should not change numeric value for .0a")
  assert(tonumber(plainUpper) == tonumber(altUpper), "# should not change numeric value for .0A")
end

print("OK")
