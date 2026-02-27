-- Bug: %g/%G without explicit precision uses Go's shortest-unique representation
-- instead of C's default 6 significant digits.
-- lua5.4: string.format("%g", 12345.6789) -> "12345.7" (6 sig digits)
-- golua:  string.format("%g", 12345.6789) -> "12345.6789" (shortest unique)

assert(string.format("%g", 12345.6789) == "12345.7",
  "expected '12345.7', got '" .. string.format("%g", 12345.6789) .. "'")

assert(string.format("%g", 123456.789) == "123457",
  "expected '123457', got '" .. string.format("%g", 123456.789) .. "'")

assert(string.format("%g", 1234567.89) == "1.23457e+06",
  "expected '1.23457e+06', got '" .. string.format("%g", 1234567.89) .. "'")

-- %G should behave the same but with uppercase E
assert(string.format("%G", 12345.6789) == "12345.7",
  "expected '12345.7', got '" .. string.format("%G", 12345.6789) .. "'")

-- With explicit precision, should work correctly
assert(string.format("%.10g", 12345.6789) == "12345.6789",
  "explicit precision should work")

-- Simple values should still be fine
assert(string.format("%g", 42) == "42",
  "simple integer should format as '42'")
assert(string.format("%g", 0.5) == "0.5",
  "simple float should format as '0.5'")

print("PASS")
