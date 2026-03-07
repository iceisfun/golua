-- Bug: string.format %a width padding zero-pads instead of space-padding
-- when the width number contains a '0' digit (e.g., %10a, %20a).
-- The zero-pad flag check matches '0' in the width digits, not just the flag position.
-- Lua 5.4 space-pads by default; only explicit %010a should zero-pad.

do
  -- %10a should space-pad (width=10, no 0 flag)
  assert(string.format("#%10a#", 1.0) == "#    0x1p+0#",
    "%10a should space-pad, got: " .. string.format("#%10a#", 1.0))

  -- %20a should space-pad (width=20, no 0 flag)
  assert(string.format("#%20a#", 1.0) == "#              0x1p+0#",
    "%20a should space-pad, got: " .. string.format("#%20a#", 1.0))

  -- %010a SHOULD zero-pad (explicit 0 flag + width=10)
  assert(string.format("#%010a#", 1.0) == "#0x00001p+0#",
    "%010a should zero-pad, got: " .. string.format("#%010a#", 1.0))

  -- %-10a left-align (no zero-pad)
  assert(string.format("#%-10a#", 1.0) == "#0x1p+0    #",
    "%-10a should left-align, got: " .. string.format("#%-10a#", 1.0))

  -- %+10a plus flag with space-pad
  assert(string.format("#%+10a#", 1.0) == "#   +0x1p+0#",
    "%+10a should space-pad with +, got: " .. string.format("#%+10a#", 1.0))

  -- %+010a plus flag with zero-pad
  assert(string.format("#%+010a#", 1.0) == "#+0x0001p+0#",
    "%+010a should zero-pad with +, got: " .. string.format("#%+010a#", 1.0))

end
