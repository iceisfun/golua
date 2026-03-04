-- Test: string.format error checking
-- From: strings.lua
-- What: Tests that string.format produces correct errors for invalid conversions, format strings that are too long, missing values, and invalid modifier combinations.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

local function check (fmt, msg)
  checkerror(msg, string.format, fmt, 10)
end

local aux = string.rep('0', 600)
check("%100.3d", "invalid conversion")
check("%1"..aux..".3d", "too long")
check("%1.100d", "invalid conversion")
check("%10.1"..aux.."004d", "too long")
check("%t", "invalid conversion")
check("%"..aux.."d", "too long")
check("%d %d", "no value")
check("%010c", "invalid conversion")
check("%.10c", "invalid conversion")
check("%0.34s", "invalid conversion")
check("%#i", "invalid conversion")
check("%3.1p", "invalid conversion")
check("%0.s", "invalid conversion")
check("%10q", "cannot have modifiers")
check("%F", "invalid conversion")   -- useless and not in C89
end
