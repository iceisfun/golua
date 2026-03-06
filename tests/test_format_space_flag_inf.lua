-- Bug: string.format space flag (' ') not applied to positive infinity.
-- Lua 5.4: "% f" on math.huge produces " inf" (space prefix).
-- GoLua: produces "inf" (no space prefix).

do
  assert(string.format("% f", math.huge) == " inf",
    "space flag on +inf with %f, got: " .. string.format("% f", math.huge))

  assert(string.format("% f", -math.huge) == "-inf",
    "space flag on -inf with %f, got: " .. string.format("% f", -math.huge))

  assert(string.format("% e", math.huge) == " inf",
    "space flag on +inf with %e, got: " .. string.format("% e", math.huge))

  assert(string.format("% g", math.huge) == " inf",
    "space flag on +inf with %g, got: " .. string.format("% g", math.huge))

  -- + flag should still produce "+inf"
  assert(string.format("%+f", math.huge) == "+inf",
    "+ flag on +inf with %f, got: " .. string.format("%+f", math.huge))

  -- NaN should not get space or + (NaN has no sign concept in Lua)
  local nan = 0/0
  assert(string.format("% f", nan) == "-nan" or string.format("% f", nan) == "nan" or string.format("% f", nan) == " nan",
    "space flag on nan, got: " .. string.format("% f", nan))

  print("PASS: format space flag on infinity")
end
