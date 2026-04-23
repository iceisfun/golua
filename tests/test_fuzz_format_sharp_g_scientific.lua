-- Fuzz regression: %#g with scientific (exp) form must pad mantissa
-- trailing zeros out to `precision` significant digits. Previously,
-- values like 1e-10 rendered as "1.e-10" instead of "1.00000000000000e-10"
-- because golua only padded under the fixed-form branch.

local function check(fmt, val, expected)
  local got = string.format(fmt, val)
  assert(got == expected,
    string.format("format(%q, %s): got %q, expected %q", fmt, tostring(val), got, expected))
end

-- Small magnitude, scientific form with '#' must pad to N sig digits
check("%#.15g", 1e-10, "1.00000000000000e-10")
check("%#.15g", 8.41093e-12, "8.41093000000000e-12")
check("%#.15g", 1.23e-7, "1.23000000000000e-07")
check("%#.6g", 1e-10, "1.00000e-10")
check("%#.3g", 1e-10, "1.00e-10")
check("%#.1g", 1e-10, "1.e-10")

-- Large magnitude, scientific form with '#' (already worked)
check("%#.15g", 1.23e20, "1.23000000000000e+20")
check("%#.3g", 1.23e10, "1.23e+10")
check("%#.2g", 9.99e10, "1.0e+11")

-- Rounding-bumped-magnitude edge case: C emits mantissa without zero
-- padding, just a trailing '.'.  These previously worked.
check("%#.2g", 99.99995, "1.e+02")
check("%#.6g", 999999.5, "1.e+06")
check("%#010.6g", 999999.5, "00001.e+06")

-- Fixed form with '#' (unchanged)
check("%#.5g", 9.999995e-05, "0.00010000")
check("%#.6g", 123.0, "123.000")

-- Non-'#' scientific form must be unchanged (shortest, no padding)
check("%.15g", 1e-10, "1e-10")
check("%.15g", 1.23e-7, "1.23e-07")

print("ok")
