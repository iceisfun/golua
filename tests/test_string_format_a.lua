-- Bug: %a/%A with precision is broken.
-- The precision is applied as string truncation (%.2s) instead of being passed
-- to the hex float formatter.
-- lua5.4: string.format("%.2a", 1.5) -> "0x1.80p+0"
-- golua:  string.format("%.2a", 1.5) -> "0x" (truncated!)

-- Basic %a without precision should work
local basic = string.format("%a", 1.5)
assert(basic:find("0x") and basic:find("p"),
  "basic %a should produce hex float, got: " .. basic)

-- %a with precision
local prec2 = string.format("%.2a", 1.5)
assert(prec2 == "0x1.80p+0",
  "expected '0x1.80p+0', got '" .. prec2 .. "'")

-- %.4a of pi
local prec4 = string.format("%.4a", math.pi)
assert(prec4 == "0x1.921fp+1",
  "expected '0x1.921fp+1', got '" .. prec4 .. "'")

-- %.0a should round
local prec0 = string.format("%.0a", 1.5)
assert(prec0 == "0x2p+0",
  "expected '0x2p+0', got '" .. prec0 .. "'")

-- %A uppercase
local upper = string.format("%A", 1.5)
assert(upper:find("0X") and upper:find("P"),
  "basic %A should produce uppercase hex float, got: " .. upper)

print("PASS")
